package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/Zolet-hash/smart-rentals/internal/database"
	"github.com/Zolet-hash/smart-rentals/internal/models"
	"github.com/gin-gonic/gin"
)

func CreateTenant(db *database.Database) gin.HandlerFunc {
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
		err := db.DB.QueryRow(
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

		err = db.DB.QueryRow(
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

		err = db.DB.QueryRow(
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

func ListAllTenants(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Get landlord ID from JWT
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

		// Query tenants belonging to landlord's units
		query := `
			SELECT 
				t.id,
				t.user_id,
				t.unit_id,
				t.created_at,
				t.updated_at
			FROM tenants t
			JOIN units u ON u.id = t.unit_id
			JOIN properties p ON p.id = u.property_id
			WHERE p.landlord_id = $1
			ORDER BY t.created_at DESC
		`

		rows, err := db.DB.Query(query, landlordID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var tenants []models.Tenant

		for rows.Next() {
			var tenant models.Tenant

			if err := rows.Scan(
				&tenant.ID,
				&tenant.UnitID,
				&tenant.CreatedAt,
				&tenant.UpdatedAt,
			); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			tenants = append(tenants, tenant)
		}

		if err := rows.Err(); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, tenants)
	}
}

func GetTenant(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		idParam := ctx.Param("id")
		tenantID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			ctx.JSON(400, gin.H{"error": "Invalid tenant id"})
			return
		}

		var tenant models.Tenant
		query := `
			SELECT
				id,
				landlord_id,
				tenant_name,
				payment_no1,
				payment_no2
				rent,
				balance,
				created_at,
				updated_at
			FROM tenants
			WHERE id = $1
		`
		err = db.DB.QueryRow(query, tenantID).Scan(
			&tenant.ID,
			&tenant.LandlordID,
			tenant.UnitID,
			&tenant.TenantName,
			&tenant.PaymentNo1,
			&tenant.PaymentNo2,
			&tenant.Rent,
			&tenant.Balance,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
		)

		if err == sql.ErrNoRows {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(200, tenant)
	}
}

func UpdateTenant(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		//Get tenant ID from URL
		tenantID := ctx.Param("id")
		if tenantID == "" {
			ctx.JSON(400, gin.H{"error": "Property ID required"})
			return
		}

		//Get landlord ID from JWT context
		userID, exists := ctx.Get("user_id")
		if !exists {
			ctx.JSON(401, gin.H{"error": "unauthorized"})
			return
		}

		landlordID, ok := userID.(float64)
		if !ok {
			ctx.JSON(401, gin.H{"error": "Invalid userID"})
		}

		//input struct partialy update
		var input struct {
			TenantName *string `json:"tenant_name"`
			PaymentNo1 *string `json:"payment_no1"`
			PaymentNo2 *string `json:"payment_no2"`
			Rent       *string `json:"rent"`
			Balance    *string `json:"balance"`
		}

		if err := ctx.ShouldBindJSON(&input); err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
		}

		query := `
			UPDATE tenants
			SET
				tenant_name = COALESCE($1, tenant_name),
				payment_no1 = COALESCE($2, payment_no1),
				payment_no2 = COALESCE($3, payment_no2)
				rent = COALESCE($4, rent),
				balance = COALESCE($5, balance)
				updated_at = NOW()
			WHERE id = $5 AND landlord_id = $6
			RETURNING id, landlord_id, tenant_name, payment_no1, payment_no2, rent, balance, updated_at
		`

		var tenant models.Tenant

		err := db.DB.QueryRow(
			query,
			input.TenantName,
			input.PaymentNo1,
			input.PaymentNo2,
			input.Rent,
			input.Balance,
			tenantID,
			int64(landlordID),
		).Scan(
			&tenant.ID,
			&tenant.TenantName,
			&tenant.PaymentNo1,
			&tenant.PaymentNo2,
			&tenant.Rent,
			&tenant.Balance,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
		)

		if err != nil {
			if err == sql.ErrNoRows {
				ctx.JSON(404, gin.H{"error": "tenant not found or not owned by user"})
				return
			}
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(200, tenant)
	}
}

// RemoveTenant removes tenant from the database, incase he vacates
func RemoveTenant(db *database.Database) gin.HandlerFunc {
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
		tx, err := db.DB.Begin()
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
		// _, err = tx.Exec(
		// 	`
		// 	UPDATE units
		// 	SET vacancy = true,
		// 	    updated_at = NOW()
		// 	WHERE id = $1
		// 	`,
		// 	unitID,
		// )
		// if err != nil {
		// 	ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		// 	return
		// }

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
