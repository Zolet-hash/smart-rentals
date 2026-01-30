package handlers

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"time"

	"github.com/Zolet-hash/smart-rentals/internal/database"
	"github.com/gin-gonic/gin"
)

type MpesaConfirmationPayload struct {
	BusinessShortCode string  `json:"BusinessShortCode"`
	BillRefNumber     string  `json:"BillRefNumber"`
	MSISDN            string  `json:"MSISDN"`
	TransAmount       float64 `json:"TransAmount"`
	TransID           string  `json:"TransID"`
	TransTime         string  `json:"TransTime"`
}

type NormalizedPayment struct {
	Provider   string
	Channel    string
	BusinessID string
	AccountRef sql.NullString
	Phone      string
	Amount     int64
	Receipt    string
}

func normalizePhone(msisdn string) string {
	// Remove all spaces, dashes, parentheses
	re := regexp.MustCompile(`[^\d\+]`)
	msisdn = re.ReplaceAllString(msisdn, "")

	// Remove leading/trailing spaces
	msisdn = strings.TrimSpace(msisdn)

	switch {
	case strings.HasPrefix(msisdn, "07"):
		return "254" + msisdn[1:]
	case strings.HasPrefix(msisdn, "01"):
		return "254" + msisdn[1:]
	case strings.HasPrefix(msisdn, "+254"):
		return strings.TrimPrefix(msisdn, "+")
	case strings.HasPrefix(msisdn, "254"):
		return msisdn
	default:
		// return as-is or return "" if invalid
		return msisdn
	}
}

func MpesaValidation(c *gin.Context) {
	c.JSON(200, gin.H{
		"ResultCode": 0,
		"ResultDesc": "Accepted",
	})
}


func MpesaPaymentConfirmation(db *database.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload MpesaConfirmationPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(400, gin.H{"ResultCode": 1})
			return
		}

		payment := NormalizedPayment{
			Provider:   "MPESA",
			BusinessID: payload.BusinessShortCode,
			Phone:      normalizePhone(payload.MSISDN),
			Amount:     int64(payload.TransAmount),
			Receipt:    payload.TransID,
		}

		// Detect channel
		if strings.TrimSpace(payload.BillRefNumber) != "" {
			payment.Channel = "PAYBILL"
			payment.AccountRef = sql.NullString{
				String: payload.BillRefNumber,
				Valid:  true,
			}
		} else {
			payment.Channel = "TILL"
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Route by channel
		switch payment.Channel {
		case "TILL":
			handleTillPayment(ctx, c, db, payment)
		case "PAYBILL":
			handlePaybillPayment(ctx, c, db, payment)
		default:
			c.JSON(200, gin.H{"ResultCode": 0})
		}
	}
}

func handleTillPayment(
	ctx context.Context,
	c *gin.Context,
	db *database.Database,
	payment NormalizedPayment,
) {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		c.JSON(500, gin.H{"ResultCode": 1})
		return
	}
	defer tx.Rollback()

	var landlordID int64
	err = tx.QueryRowContext(ctx, `
		SELECT landlord_id
		FROM tills
		WHERE till_number = $1 AND active = true
	`, payment.BusinessID).Scan(&landlordID)

	if err != nil {
		// Unknown till → accept silently
		c.JSON(200, gin.H{"ResultCode": 0})
		return
	}

	// Idempotency check
	var exists bool
	_ = tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM till_transactions WHERE receipt_number = $1
		)
	`, payment.Receipt).Scan(&exists)

	if exists {
		c.JSON(200, gin.H{"ResultCode": 0})
		return
	}

	// Match tenant
	var tenantID sql.NullInt64
	_ = tx.QueryRowContext(ctx, `
		SELECT id
		FROM tenants
		WHERE landlord_id = $1
		  AND $2 IN (payment_no1, payment_no2)
		LIMIT 1
	`, landlordID, payment.Phone).Scan(&tenantID)

	// Insert transaction
	_, err = tx.ExecContext(ctx, `
		INSERT INTO till_transactions
			(receipt_number, till_number, landlord_id, tenant_id,
			 phone_number, amount)
		VALUES ($1,$2,$3,$4,$5,$6)
	`,
		payment.Receipt,
		payment.BusinessID,
		landlordID,
		tenantID,
		payment.Phone,
		payment.Amount,
	)

	if err != nil {
		c.JSON(500, gin.H{"ResultCode": err.Error()})
		return
	}

	// Update tenant balance
	if tenantID.Valid {
		_, err = tx.ExecContext(ctx, `
			UPDATE tenants
			SET balance = balance - $1,
			    updated_at = now()
			WHERE id = $2
		`, payment.Amount, tenantID.Int64)

		if err != nil {
			c.JSON(500, gin.H{"ResultCode": 1})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(500, gin.H{"ResultCode": 1})
		return
	}

	c.JSON(200, gin.H{
		"ResultCode": 0,
		"ResultDesc": "Accepted",
	})
}

func handlePaybillPayment(
	ctx context.Context,
	c *gin.Context,
	db *database.Database,
	payment NormalizedPayment,
) {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		c.JSON(500, gin.H{"ResultCode": 1})
		return
	}
	defer tx.Rollback()

	var landlordID int64
	err = tx.QueryRowContext(ctx, `
		SELECT landlord_id
		FROM paybills
		WHERE paybill = $1
		  AND account_number = $2
		  AND active = true
	`,
		payment.BusinessID,
		payment.AccountRef.String,
	).Scan(&landlordID)

	if err != nil {
		c.JSON(200, gin.H{"ResultCode": 0})
		return
	}

	// Idempotency
	var exists bool
	_ = tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM paybill_transactions WHERE receipt_number = $1
		)
	`, payment.Receipt).Scan(&exists)

	if exists {
		c.JSON(200, gin.H{"ResultCode": 0})
		return
	}

	// Match tenant
	var tenantID sql.NullInt64
	_ = tx.QueryRowContext(ctx, `
		SELECT id
		FROM tenants
		WHERE landlord_id = $1
		  AND $2 IN (payment_no1, payment_no2)
		LIMIT 1
	`, landlordID, payment.Phone).Scan(&tenantID)

	matched := tenantID.Valid

	_, err = tx.ExecContext(ctx, `
		INSERT INTO paybill_transactions
			(receipt_number, paybill, account_number, landlord_id,
			 tenant_id, phone_number, amount)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`,
		payment.Receipt,
		payment.BusinessID,
		payment.AccountRef.String,
		landlordID,
		tenantID,
		payment.Phone,
		payment.Amount,
	)

	if err != nil {
		c.JSON(500, gin.H{"ResultCode": 1})
		return
	}

	if matched {
		_, err = tx.ExecContext(ctx, `
			UPDATE tenants
			SET balance = balance - $1,
			    updated_at = now()
			WHERE id = $2
		`, payment.Amount, tenantID.Int64)

		if err != nil {
			c.JSON(500, gin.H{"ResultCode": 1})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(500, gin.H{"ResultCode": 1})
		return
	}

	c.JSON(200, gin.H{
		"ResultCode": 0,
		"ResultDesc": "Accepted",
	})
}

// Till (C2B)
// {
//   "BusinessShortCode": "123456", -> till number
//   "TransID": "QKJ7",
//   "MSISDN": "254712345678", -> phone number
//   "TransAmount": 10000
// }

// PayBill (C2B)
// {
//   "BusinessShortCode": "987654", -> paybill no
//   "TransID": "XYZ9",
//   "MSISDN": "254722000111", -> phone
//   "TransAmount": 12000,
//   "BillRefNumber": "A12" -> account
// }
