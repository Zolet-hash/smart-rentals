package handlers

import (
	"math"
	"net/http"
	"strconv"

	"github.com/Zolet-hash/smart-rentals/internal/database"
	"github.com/Zolet-hash/smart-rentals/internal/models"
	"github.com/gin-gonic/gin"
)

func CreateUnit(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var unit models.Unit

		// Bind JSON request body
		if err := ctx.ShouldBindJSON(&unit); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		// var exists bool
		// err := db.QueryRow(
		// 	"SELECT EXISTS (SELECT 1 FROM properties WHERE id = $1)",
		// 	unit.PropertyID,
		// ).Scan(&exists)

		// if err != nil || !exists {
		// 	ctx.JSON(http.StatusBadRequest, gin.H{
		// 		"error": "property does not exist",
		// 	})
		// 	return
		// }

		// Basic validation
		if unit.PropertyID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "property_id is required",
			})
			return
		}

		if unit.UnitPrice <= 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "unit_price must be greater than zero",
			})
			return
		}

		query := `
			INSERT INTO units (
				property_id,
				unit_name,
				unit_type,
				unit_price,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
			RETURNING id, created_at, updated_at
		`

		err := db.DB.QueryRow(
			query,
			unit.PropertyID,
			unit.UnitName,
			unit.UnitType,
			unit.UnitPrice,
		).Scan(
			&unit.ID,
			&unit.CreatedAt,
			&unit.UpdatedAt,
		)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		ctx.JSON(http.StatusCreated, unit)
	}
}

func GetUnitsByProperty(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Parse property ID
		idParam := ctx.Param("id")
		propertyID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid property id",
			})
			return
		}

		// Pagination defaults
		page := 1
		limit := 10

		if p := ctx.Query("page"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}

		if l := ctx.Query("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
				limit = v
			}
		}

		offset := (page - 1) * limit

		// Fetch units
		query := `
			SELECT
				id,
				property_id,
				unit_name,
				unit_type,
				unit_price,
				created_at,
				updated_at
			FROM units
			WHERE property_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`

		rows, err := db.DB.Query(query, propertyID, limit, offset)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		defer rows.Close()

		units := make([]models.Unit, 0)

		for rows.Next() {
			var unit models.Unit

			if err := rows.Scan(
				&unit.ID,
				&unit.PropertyID,
				&unit.UnitName,
				&unit.UnitType,
				&unit.UnitPrice,
				&unit.CreatedAt,
				&unit.UpdatedAt,
			); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			units = append(units, unit)
		}

		// Total count (for pagination metadata)
		var total int
		countQuery := `
			SELECT COUNT(*)
			FROM units
			WHERE property_id = $1
		`
		if err := db.DB.QueryRow(countQuery, propertyID).Scan(&total); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		totalPages := int(math.Ceil(float64(total) / float64(limit)))

		ctx.JSON(http.StatusOK, gin.H{
			"data": units,
			"pagination": gin.H{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": totalPages,
			},
		})
	}
}
