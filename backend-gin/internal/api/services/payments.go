package services

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/Zolet-hash/smart-rentals/internal/database"
	"github.com/gin-gonic/gin"
)

type PropertyCharge struct {
	PropertyID uint `json:"property_id"`
	UnitCount  int  `json:"unit_count"`
	Charge     int  `json:"charge"`
}

type LandlordBillingSummary struct {
	PropertyCount int              `json:"property_count"`
	Properties    []PropertyCharge `json:"properties"`
	TotalAmount   int              `json:"total_amount"`
}

func GetLandlordBilling(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {

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

		summary, err := CalculateLandlordBilling(
			ctx.Request.Context(),
			db.DB,
			landlordID,
		)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, summary)
	}
}

func CalculateLandlordBilling(
	ctx context.Context,
	db *sql.DB,
	landlordID uint,
) (*LandlordBillingSummary, error) {

	query := `
		SELECT 
			p.id AS property_id,
			COUNT(u.id) AS unit_count,
			CASE
				WHEN COUNT(u.id) <= 10 THEN 700
				WHEN COUNT(u.id) > 10 AND COUNT(u.id) <= 20 THEN 1000
				ELSE 1200
			END AS charge
		FROM properties p
		LEFT JOIN units u ON u.property_id = p.id
		WHERE p.landlord_id = $1
		GROUP BY p.id
		ORDER BY p.id
	`

	rows, err := db.QueryContext(ctx, query, landlordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summary LandlordBillingSummary
	var total int

	for rows.Next() {
		var item PropertyCharge

		err := rows.Scan(
			&item.PropertyID,
			&item.UnitCount,
			&item.Charge,
		)
		if err != nil {
			return nil, err
		}

		total += item.Charge
		summary.Properties = append(summary.Properties, item)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	summary.PropertyCount = len(summary.Properties)
	summary.TotalAmount = total

	return &summary, nil
}
