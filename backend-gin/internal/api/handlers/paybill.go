package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Zolet-hash/smart-rentals/internal/database"
	"github.com/gin-gonic/gin"
)

type CreatePaybillRequest struct {
	Paybill       string `json:"paybill" binding:"required"`
	AccountNumber string `json:"account_number" binding:"required"`
	DarajaAppID   int64  `json:"daraja_app_id" binding:"required"`
}

func CreatePaybill(db *database.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreatePaybillRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()}) //to change in prod
			return
		}

		landlordID := c.GetInt64("user_id") // assuming auth middleware

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tx, err := db.DB.BeginTx(ctx, nil)
		if err != nil {
			c.JSON(500, gin.H{"error": "db error"})
			return
		}
		defer tx.Rollback()

		var paybillID int64
		var callbackRegistered bool

		err = tx.QueryRowContext(ctx, `
			INSERT INTO paybills
				(paybill, account_number, landlord_id, daraja_app_id)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT (paybill, account_number) DO UPDATE
			SET active = true
			RETURNING id, callback_registered
		`,
			req.Paybill,
			req.AccountNumber,
			landlordID,
			req.DarajaAppID,
		).Scan(&paybillID, &callbackRegistered)

		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()}) //I'll change in prod
			return
		}

		//Register callbacks ONCE per shortcode
		if !callbackRegistered {
			err = RegisterDarajaCallbacks(ctx, tx, req.DarajaAppID, req.Paybill)
			if err != nil {
				c.JSON(502, gin.H{"error": "failed to register callbacks"})
				return
			}

			_, _ = tx.ExecContext(ctx, `
				UPDATE paybills
				SET callback_registered = true
				WHERE id = $1
			`, paybillID)
		}

		if err := tx.Commit(); err != nil {
			c.JSON(500, gin.H{"error": "commit failed"})
			return
		}

		c.JSON(201, gin.H{
			"id":         paybillID,
			"paybill":    req.Paybill,
			"account":    req.AccountNumber,
			"registered": true,
		})
	}
}

func RegisterDarajaCallbacks(
	ctx context.Context,
	tx *sql.Tx,
	darajaAppID int64,
	shortcode string,
) error {

	var creds struct {
		ConsumerKey    string
		ConsumerSecret string
	}

	err := tx.QueryRowContext(ctx, `
		SELECT consumer_key, consumer_secret
		FROM daraja_apps
		WHERE id = $1
	`, darajaAppID).Scan(&creds.ConsumerKey, &creds.ConsumerSecret)

	if err != nil {
		return err
	}

	accessToken, err := GetDarajaAccessToken(creds.ConsumerKey, creds.ConsumerSecret)
	if err != nil {
		return err
	}

	payload := map[string]string{
		"ShortCode":       shortcode,
		"ResponseType":    "Completed",
		"ConfirmationURL": "https://api.yourapp.com/mpesa/confirmation",
		"ValidationURL":   "https://api.yourapp.com/mpesa/validation",
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(
		"POST",
		"https://api.safaricom.co.ke/mpesa/c2b/v1/registerurl",
		bytes.NewBuffer(body),
	)

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("daraja returned %d", resp.StatusCode)
	}

	return nil
}

// GetDarajaAccessToken obtains consumerKey and consmerSecret from daraja
func GetDarajaAccessToken(consumerKey, consumerSecret string) (string, error) {
	req, _ := http.NewRequest(
		"GET",
		"https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials",
		nil,
	)

	req.SetBasicAuth(consumerKey, consumerSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.AccessToken, nil
}
