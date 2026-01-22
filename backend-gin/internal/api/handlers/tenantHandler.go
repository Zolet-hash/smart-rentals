package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/Zolet-hash/smart-rentals/internal/models"
	"github.com/gin-gonic/gin"
)

func CreateTenant(db *sql.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var tenant models.Tenant

		// Bind request body
		if err := ctx.ShouldBindJSON(&tenant); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate required fields
		if tenant.ID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}

		if tenant.UnitID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "unit_id is required"})
			return
		}

		//Get landlord ID from JWT
		userID, exists := ctx.Get("user_id")
		if !exists {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		landlordIDFloat, ok := userID.(float64)
		if !ok || landlordIDFloat <= 0 {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
			return
		}

		landlordID := uint(landlordIDFloat)

		//Ensure user(Landlord) exists
		var userExists bool
		err := db.QueryRow(
			`SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`,
			tenant.ID,
		).Scan(&userExists)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if !userExists {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Landlord does not exist"})
			return
		}

		// Verify landlord owns the unit (via property)
		var ownsUnit bool
		ownershipQuery := `
			SELECT EXISTS (
				SELECT 1
				FROM units u
				JOIN properties p ON p.id = u.property_id
				WHERE u.id = $1 AND p.landlord_id = $2
			)
		`

		err = db.QueryRow(
			ownershipQuery,
			tenant.UnitID,
			landlordID,
		).Scan(&ownsUnit)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if !ownsUnit {
			ctx.JSON(http.StatusForbidden, gin.H{
				"error": "You do not own this unit",
			})
			return
		}

		//Assign tenant to unit
		query := `
			INSERT INTO tenants (
				user_id,
				unit_id,
				created_at,
				updated_at
			)
			VALUES ($1, $2, NOW(), NOW())
			RETURNING id, created_at, updated_at
		`

		err = db.QueryRow(
			query,
			tenant.ID,
			tenant.UnitID,
		).Scan(
			&tenant.ID,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
		)

		if err != nil {
			// Unique violation (user already assigned)
			if strings.Contains(err.Error(), "unique") {
				ctx.JSON(http.StatusConflict, gin.H{
					"error": "tenant already assigned to a unit",
				})
				return
			}

			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusCreated, tenant)
	}
}

func RemoveTenant(db *sql.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tenantID := ctx.Param("id")
		if tenantID == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "tenant id is required"})
			return
		}

		//Get landlord ID from JWT
		userID, exists := ctx.Get("user_id")
		if !exists {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		landlordIDFloat, ok := userID.(float64)
		if !ok || landlordIDFloat <= 0 {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
			return
		}

		landlordID := uint(landlordIDFloat)

		// Begin transaction
		tx, err := db.Begin()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback()

		//Get unit_id & verify ownership
		var unitID uint
		ownershipQuery := `
			SELECT t.unit_id
			FROM tenants t
			JOIN units u ON u.id = t.unit_id
			JOIN properties p ON p.id = u.property_id
			WHERE t.id = $1 AND p.landlord_id = $2
		`

		err = tx.QueryRow(
			ownershipQuery,
			tenantID,
			landlordID,
		).Scan(&unitID)

		if err != nil {
			if err == sql.ErrNoRows {
				ctx.JSON(http.StatusNotFound, gin.H{
					"error": "tenant not found or not authorized",
				})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		//Delete tenant
		_, err = tx.Exec(
			`DELETE FROM tenants WHERE id = $1`,
			tenantID,
		)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		//Mark unit as vacant
		_, err = tx.Exec(
			`
			UPDATE units
			SET vacancy = true,
			    updated_at = NOW()
			WHERE id = $1
			`,
			unitID,
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

		ctx.JSON(http.StatusOK, gin.H{
			"message": "tenant removed and unit marked vacant",
		})
	}
}
