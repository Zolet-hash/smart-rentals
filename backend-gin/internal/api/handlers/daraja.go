package handlers

import (
	"context"
	"database/sql"
	"time"

	"github.com/Zolet-hash/smart-rentals/internal/database"
	"github.com/gin-gonic/gin"
)

type NormalizedPayment struct {
	Provider   string
	Channel    string
	BusinessID string
	AccountRef sql.NullString
	Phone      string
	Amount     int64
	Receipt    string
}


func MpesaPaymentConfirmation(db *database.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload map[string]interface{}
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(400, gin.H{"ResultCode": 1})
			return
		}

		businessID := payload["BusinessShortCode"].(string) //either till or paybill

		payment := NormalizedPayment{
			Provider:   "MPESA",
			BusinessID: businessID,
			Phone:      payload["MSISDN"].(string),
			Amount:     int64(payload["TransAmount"].(float64)),
			Receipt:    payload["TransID"].(string),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		//Channel detection
		if ref, ok := payload["BillRefNumber"]; ok && ref != "" {
			payment.Channel = "PAYBILL"
			payment.AccountRef = sql.NullString{
				String: ref.(string),
				Valid:  true,
			}
		} else {
			payment.Channel = "TILL"
		}

		//Resolve landord from till
		if payment.Channel == "Till" {
			var landlordID int64
			err := db.DB.QueryRowContext(ctx, `
			SELECT landlord_id FROM tills
			WHERE till_number = $1 AND active = true
			`, payment.Channel).Scan(&landlordID)

			if err != nil {
				//Unkmown till -> accept but store for audit
				c.JSON(200, gin.H{"ResultCode": 0})
				return
			}

			//Idemptotency check -> network error, transactions duplicate
			var exists bool
			_ = db.DB.QueryRowContext(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM till_transactions WHERE receipt_number = $1
				)
			`, payment.Receipt).Scan(&exists)

			if exists {
				c.JSON(200, gin.H{"ResultCode": 0})
			}

			//match tenant within landlord
			var tenantID sql.NullInt64
			err = db.DB.QueryRowContext(ctx, `
				SELECT id FROM tenants
				WHERE landlord_id = $1
					AND $2 IN (payment_no1, payment_no2)
				LIMIT 1
			`, landlordID, payment.Phone).Scan(&tenantID)

			matched := tenantID.Valid

			//Insert transaction
			_, err = db.DB.ExecContext(ctx, `
				INSERT INTO till_transactions
				(receipt_number, till_number, landlord_id, tenant_id,
				phone_number, amount, trans_time)
				VALUES ($1,$2,$3,$4,$5,$6,$7,now())
			`,
				payment.Receipt,
				payment.BusinessID,
				landlordID,
				tenantID,
				payment.Phone,
				payment.Amount,
			)

			if err != nil {
				c.JSON(500, gin.H{"ResultCode": 1})
				return
			}

			//Update balance safely
			if matched {
				_, _ = db.DB.ExecContext(ctx, `
				UPDATE tenants
				SET balance = balance - $1,
					updated_at = now()
				WHERE id = $2
			`, payment.Amount, tenantID.Int64)
			}

			c.JSON(200, gin.H{
				"ResultCode": 0,
				"ResultDesc": "Accepted",
			})
		}
		//else -> if paymentChannel is paybill
		var landlordID int64
		err := db.DB.QueryRowContext(ctx, `
			SELECT landlord_id FROM paybills
			WHERE paybill = $1 AND account_number = $2 AND active = true
		`, payment.BusinessID, payment.AccountRef).Scan(&landlordID)

		if err != nil {
			// unknown paybill -> accept but store for audit
			c.JSON(200, gin.H{"ResultCode": 0})
			return
		}

		//Idempotency check
		var exists bool
		_ = db.DB.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM paybill_transactions WHERE receipt_number = $1
			)
		`, payment.Receipt).Scan(&exists)

		if exists {
			c.JSON(200, gin.H{"ResultCode": 0})
			return
		}

		//match tenant within landlord
		var tenantID sql.NullInt64
		err = db.DB.QueryRowContext(ctx, `
			SELECT id FROM tenants
			WHERE landlord_id = $1
				AND $2 IN (payment_no1, payment_no2)
			LIMIT 1
		`, landlordID, payment.Phone).Scan(&tenantID)

		matched := tenantID.Valid

		//Insert paybill transaction
		_, err = db.DB.ExecContext(ctx, `
			INSERT INTO paybill_transactions
			(receipt_number, paybill, account_number, landlord_id, tenant_id,
			phone_number, amount, matched, trans_time)
			VALUES ($1,$2,$3,$4,$5,$6,$7, now())
		`,
		payment.Receipt,
		payment.BusinessID,
		payment.AccountRef,
		landlordID,
		tenantID,
		payment.Phone,
		payment.Amount,
		matched,
		)

		if err != nil {
			c.JSON(500, gin.H{"ResultCode": 1})
			return 
		}

		//Update balance gracefuly
		if matched {
			_,_ = db.DB.ExecContext(ctx, `
				UPDATE tenants
					SET balance = balance - $1,
				WHERE id = $2
			`, payment.Amount, tenantID.Int64)
		}

		c.JSON(200, gin.H{
			"ResultCode": 0,
			"ResultDesc": "Accepted",
		})

	}
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
