package handlers

import (
	"database/sql"
	"math"
	"net/http"
	"strconv"

	"github.com/Zolet-hash/smart-rentals/internal/database"
	"github.com/Zolet-hash/smart-rentals/internal/models"
	"github.com/gin-gonic/gin"
)

// Creates a new property
func CreateProperty(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var property models.Property

		// Bind JSON (landlord_id is NOT expected from client)
		if err := ctx.ShouldBindJSON(&property); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get landlord ID from JWT context
		userID, exists := ctx.Get("user_id")
		if !exists {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// JWT numeric claims are float64
		landlordID, ok := userID.(float64)
		if !ok {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
			return
		}

		// Set landlord ID from token (override anything client sent)
		property.LandlordID = uint(landlordID)

		query := `
			INSERT INTO properties (
				landlord_id,
				title,
				description,
				location,
				property_type,
				vacancy,
				total_rent,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
			RETURNING id, created_at, updated_at
		`

		err := db.DB.QueryRow(
			query,
			property.LandlordID,
			property.Title,
			property.Description,
			property.Location,
			property.PropertyType,
			property.Vacancy,
			property.TotalRent,
		).Scan(
			&property.ID,
			&property.CreatedAt,
			&property.UpdatedAt,
		)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusCreated, property)
	}
}

// Gets a property by property id
func GetProperty(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		idParam := ctx.Param("id")
		propertyID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property id"})
			return
		}

		var property models.Property
		query := `
			SELECT 
				id,
				landlord_id,
				title,
				description,
				location,
				property_type,
				vacancy,
				total_rent,
				created_at,
				updated_at
			FROM properties
			WHERE id = $1
		`

		err = db.DB.QueryRow(query, propertyID).Scan(
			&property.ID,
			&property.LandlordID,
			&property.Title,
			&property.Description,
			&property.Location,
			&property.PropertyType,
			&property.Vacancy,
			&property.TotalRent,
			&property.CreatedAt,
			&property.UpdatedAt,
		)

		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
			return
		}
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, property)
	}
}

// Gets all properties,, pagination included
func ListProperties(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Defaults
		page := 1
		limit := 10

		// Query params
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

		// Query properties
		query := `
			SELECT
				id,
				landlord_id,
				title,
				description,
				location,
				property_type,
				vacancy,
				total_rent,
				created_at,
				updated_at
			FROM properties
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`

		rows, err := db.DB.Query(query, limit, offset)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		defer rows.Close()

		properties := make([]models.Property, 0)

		for rows.Next() {
			var property models.Property

			if err := rows.Scan(
				&property.ID,
				&property.LandlordID,
				&property.Title,
				&property.Description,
				&property.Location,
				&property.PropertyType,
				&property.Vacancy,
				&property.TotalRent,
				&property.CreatedAt,
				&property.UpdatedAt,
			); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			properties = append(properties, property)
		}

		// Count total records (for pagination metadata)
		var total int
		countQuery := `SELECT COUNT(*) FROM properties`
		if err := db.DB.QueryRow(countQuery).Scan(&total); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		// Pagination metadata
		totalPages := int(math.Ceil(float64(total) / float64(limit)))

		ctx.JSON(http.StatusOK, gin.H{
			"data": properties,
			"pagination": gin.H{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": totalPages,
			},
		})
	}
}

// Todo
// func to update propery
func UpdateProperty(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Get property ID from URL
		propertyID := ctx.Param("id")
		if propertyID == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Property ID required"})
			return
		}

		// Get landlord ID from JWT context
		userID, exists := ctx.Get("user_id")
		if !exists {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		landlordID, ok := userID.(float64) // JWT numbers decode as float64
		if !ok {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
			return
		}

		// Input struct partialy update(PATCH)
		var input struct {
			Title        *string  `json:"title"`
			Description  *string  `json:"description"`
			Location     *string  `json:"location"`
			PropertyType *string  `json:"property_type"`
			Vacancy      *bool    `json:"vacancy"`
			TotalRent    *float64 `json:"total_rent"`
		}

		if err := ctx.ShouldBindJSON(&input); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		query := `
			UPDATE properties
			SET
				title = COALESCE($1, title),
				description = COALESCE($2, description),
				location = COALESCE($3, location),
				property_type = COALESCE($4, property_type),
				vacancy = COALESCE($5, vacancy),
				total_rent = COALESCE($6, total_rent),
				updated_at = NOW()
			WHERE id = $7 AND landlord_id = $8
			RETURNING id, landlord_id, title, description, location, property_type,
			          vacancy, total_rent, created_at, updated_at
		`

		var property models.Property

		err := db.DB.QueryRow(
			query,
			input.Title,
			input.Description,
			input.Location,
			input.PropertyType,
			input.Vacancy,
			input.TotalRent,
			propertyID,
			int64(landlordID),
		).Scan(
			&property.ID,
			&property.LandlordID,
			&property.Title,
			&property.Description,
			&property.Location,
			&property.PropertyType,
			&property.Vacancy,
			&property.TotalRent,
			&property.CreatedAt,
			&property.UpdatedAt,
		)

		if err != nil {
			if err == sql.ErrNoRows {
				ctx.JSON(http.StatusNotFound, gin.H{
					"error": "Property not found or not owned by user",
				})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, property)
	}
}
