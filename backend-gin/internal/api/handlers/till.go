package handlers

import (
	"context"
	"time"

	"github.com/Zolet-hash/smart-rentals/internal/database"
	"github.com/gin-gonic/gin"
)

type CreateTillRequest struct {
	TillNumber  string `json:"till_number" binding:"required"`
	DarajaAppID int64  `json:"daraja_app_id" binding:"required"`
}

func CreateTill(db *database.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateTillRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		landlordID := c.GetInt64("user_id") // from auth middleware

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tx, err := db.DB.BeginTx(ctx, nil)
		if err != nil {
			c.JSON(500, gin.H{"error": "db error"})
			return
		}
		defer tx.Rollback()

		var tillID int64
		var callbackRegistered bool

		err = tx.QueryRowContext(ctx, `
			INSERT INTO tills
				(till_number, landlord_id, daraja_app_id)
			VALUES ($1,$2,$3)
			ON CONFLICT (till_number) DO UPDATE
			SET active = true
			RETURNING id, callback_registered
		`,
			req.TillNumber,
			landlordID,
			req.DarajaAppID,
		).Scan(&tillID, &callbackRegistered)

		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		// 🔑 Register callbacks ONCE per till number
		if !callbackRegistered {
			err = RegisterDarajaCallbacks(ctx, tx, req.DarajaAppID, req.TillNumber)
			if err != nil {
				c.JSON(502, gin.H{"error": "failed to register callbacks"})
				return
			}

			_, _ = tx.ExecContext(ctx, `
				UPDATE tills
				SET callback_registered = true
				WHERE id = $1
			`, tillID)
		}

		if err := tx.Commit(); err != nil {
			c.JSON(500, gin.H{"error": "commit failed"})
			return
		}

		c.JSON(201, gin.H{
			"id":         tillID,
			"till":       req.TillNumber,
			"registered": true,
		})
	}
}
