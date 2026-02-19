package services

import (
	"context"
	"database/sql"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Zolet-hash/smart-rentals/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type SendMoneyRequest struct {
	Message string `json:"message" binding:"required"`
}

var (
	mpesaCodeRe = regexp.MustCompile(`^([A-Z0-9]{10})`)
	amountRe    = regexp.MustCompile(`Ksh\s?([\d,]+\.\d{2})`)
	fromRe      = regexp.MustCompile(`from\s+(.+?)\s+(\+?\d{9,15})\s+on`)
)

func RecordSendMoney(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req SendMoneyRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		// Get landlord ID from JWT
		userID, exists := ctx.Get("user_id")
		if !exists {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		landlordIDFloat, ok := userID.(float64)
		if !ok || landlordIDFloat <= 0 {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
			return
		}
		landlordID := uint(landlordIDFloat)

		msg := strings.TrimSpace(req.Message)

		// Parse M-Pesa message
		codeMatch := mpesaCodeRe.FindStringSubmatch(msg)
		if len(codeMatch) < 2 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid mpesa message (code)"})
			return
		}
		mpesaCode := codeMatch[1]

		amountMatch := amountRe.FindStringSubmatch(msg)
		if len(amountMatch) < 2 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid mpesa message (amount)"})
			return
		}
		amountStr := strings.ReplaceAll(amountMatch[1], ",", "")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount <= 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
			return
		}

		fromMatch := fromRe.FindStringSubmatch(msg)
		if len(fromMatch) < 3 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid mpesa message (sender)"})
			return
		}

		senderName := strings.TrimSpace(fromMatch[1])
		phone := NormalizePhone(strings.TrimSpace(fromMatch[2]))

		// Begin transaction
		tx, err := db.DB.Begin()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		committed := false
		defer func() {
			if !committed {
				tx.Rollback()
			}
		}()

		// Insert send_money
		var recordID uint
		var createdAt time.Time

		insertQuery := `
			INSERT INTO send_money (
				landlord_id,
				mpesa_code,
				sender_name,
				amount,
				phone,
				created_at
			)
			VALUES ($1, $2, $3, $4, $5, NOW())
			RETURNING id, created_at
		`

		err = tx.QueryRow(
			insertQuery,
			landlordID,
			mpesaCode,
			senderName,
			amount,
			phone,
		).Scan(&recordID, &createdAt)

		if err != nil {
			if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
				ctx.JSON(http.StatusConflict, gin.H{"error": "payment already recorded"})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Auto-match payment to tenant
		tenantID, matched, err := MatchSendMoneyToTenant(
			ctx.Request.Context(),
			tx,
			recordID,
		)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// apply paid amount intenant balance
		newBalance, err := ApplySendMoneyToTenantBalance(
			ctx.Request.Context(),
			tx,
			recordID,
		)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		committed = true

		// Response
		ctx.JSON(http.StatusCreated, gin.H{
			"id":          recordID,
			"mpesa_code":  mpesaCode,
			"amount":      amount,
			"sender":      senderName,
			"phone":       phone,
			"matched":     matched,
			"tenant_id":   tenantID,
			"new_balance": newBalance,
			"created_at":  createdAt,
		})
	}
}

func MatchSendMoneyToTenant(
	ctx context.Context,
	tx *sql.Tx,
	sendMoneyID uint,
) (uint, bool, error) {

	var tenantID uint

	query := `
		UPDATE send_money sm
		SET
			tenant_id = t.id,
			matched   = true
		FROM tenants t
		WHERE sm.id = $1
		  AND sm.matched = false
		  AND sm.landlord_id = t.landlord_id
		  AND (
				sm.phone = t.payment_no1
			 OR sm.phone = t.payment_no2
		  )
		RETURNING t.id
	`

	err := tx.QueryRowContext(ctx, query, sendMoneyID).Scan(&tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			// No match found (not an error)
			return 0, false, nil
		}
		return 0, false, err
	}

	return tenantID, true, nil
}

func ApplySendMoneyToTenantBalance(
	ctx context.Context,
	tx *sql.Tx,
	sendMoneyID uint,
) (float64, error) {

	var tenantID uint
	var amount float64

	// Lock the send_money row to avoid double application
	query := `
		SELECT tenant_id, amount
		FROM send_money
		WHERE id = $1
		  AND matched = true
		  AND applied = false
		FOR UPDATE
	`

	err := tx.QueryRowContext(ctx, query, sendMoneyID).
		Scan(&tenantID, &amount)

	if err != nil {
		if err == sql.ErrNoRows {
			// Not matched or already applied — not an error
			return 0, nil
		}
		return 0, err
	}

	// Subtract payment from tenant balance (negative allowed)
	var newBalance float64
	err = tx.QueryRowContext(
		ctx,
		`
		UPDATE tenants
		SET balance = balance - $1,
		    updated_at = NOW()
		WHERE id = $2
		RETURNING balance
		`,
		amount,
		tenantID,
	).Scan(&newBalance)

	if err != nil {
		return 0, err
	}

	// Mark payment as applied
	_, err = tx.ExecContext(
		ctx,
		`
		UPDATE send_money
		SET applied = true
		WHERE id = $1
		`,
		sendMoneyID,
	)

	if err != nil {
		return 0, err
	}

	return newBalance, nil
}
