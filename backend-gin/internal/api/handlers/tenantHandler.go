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
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		// Validate required fields
		if tenant.UserID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "user_id is required",
			})
			return
		}

		// Optional: ensure user exists
		var exists bool
		err := db.QueryRow(
			`SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`,
			tenant.UserID,
		).Scan(&exists)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		if !exists {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "user does not exist",
			})
			return
		}

		// Insert tenant
		query := `
			INSERT INTO tenants (user_id, created_at, updated_at)
			VALUES ($1, NOW(), NOW())
			RETURNING id, created_at, updated_at
		`

		err = db.QueryRow(query, tenant.UserID).Scan(
			&tenant.ID,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
		)

		if err != nil {
			// Unique violation (user already a tenant)
			if strings.Contains(err.Error(), "unique") {
				ctx.JSON(http.StatusConflict, gin.H{
					"error": "tenant already exists for this user",
				})
				return
			}

			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		ctx.JSON(http.StatusCreated, tenant)
	}
}
