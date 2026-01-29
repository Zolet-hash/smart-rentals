package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

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

		tenant.PaymentNo1 = normalizePhone(tenant.PaymentNo1)
		tenant.PaymentNo2 = normalizePhone(tenant.PaymentNo2)

		if tenant.UnitID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "unit_id is required"})
			return
		}

		if tenant.Rent <= 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "rent must be greater than zero"})
			return
		}

		// Get landlord ID from JWT
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

		// Begin transaction
		tx, err := db.DB.Begin()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		committed := false
		defer func() {
			if !committed {
				tx.Rollback()
			}
		}()

		// Verify landlord owns the unit AND unit is vacant
		var isVacant bool
		ownershipQuery := `
			SELECT u.vacancy
			FROM units u
			JOIN properties p ON p.id = u.property_id
			WHERE u.id = $1 AND p.landlord_id = $2
		`

		err = tx.QueryRow(
			ownershipQuery,
			tenant.UnitID,
			landlordID,
		).Scan(&isVacant)

		if err != nil {
			if err == sql.ErrNoRows {
				ctx.JSON(http.StatusForbidden, gin.H{"error": "unit not found or not owned by user"})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if !isVacant {
			ctx.JSON(http.StatusConflict, gin.H{"error": "unit is already occupied"})
			return
		}

		// Insert tenant (balance handled by trigger)
		insertTenantQuery := `
			INSERT INTO tenants (
				tenant_name,
				payment_no1,
				payment_no2,
				unit_id,
				landlord_id,
				rent,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
			RETURNING id, balance, created_at, updated_at
		`

		err = tx.QueryRow(
			insertTenantQuery,
			tenant.TenantName,
			tenant.PaymentNo1,
			tenant.PaymentNo2,
			tenant.UnitID,
			landlordID,
			tenant.Rent,
		).Scan(
			&tenant.ID,
			&tenant.Balance,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
		)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Mark unit as occupied
		_, err = tx.Exec(
			`
			UPDATE units
			SET vacancy = false,
			    updated_at = NOW()
			WHERE id = $1
			`,
			tenant.UnitID,
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
		committed = true

		tenant.LandlordID = landlordID

		ctx.JSON(http.StatusCreated, tenant)
	}
}

func ListAllTenants(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Get landlord ID from JWT
		userID, exists := ctx.Get("user_id")
		if !exists {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		landlordID := uint(userID.(float64))

		query := `
			SELECT
				t.id,
				t.tenant_name,
				t.payment_no1,
				t.payment_no2,
				t.rent,
				t.balance,
				t.unit_id,
				u.unit_name,
				u.unit_type,
				u.unit_price,
				t.created_at,
				t.updated_at
			FROM tenants t
			JOIN units u ON u.id = t.unit_id
			WHERE t.landlord_id = $1
			ORDER BY t.created_at DESC
		`

		rows, err := db.DB.Query(query, landlordID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var tenants []models.TenantWithUnit

		for rows.Next() {
			var tenant models.TenantWithUnit

			if err := rows.Scan(
				&tenant.ID,
				&tenant.TenantName,
				&tenant.PaymentNo1,
				&tenant.PaymentNo2,
				&tenant.Rent,
				&tenant.Balance,
				&tenant.UnitID,
				&tenant.UnitName,
				&tenant.UnitType,
				&tenant.UnitPrice,
				&tenant.CreatedAt,
				&tenant.UpdatedAt,
			); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			tenants = append(tenants, tenant)
		}

		ctx.JSON(http.StatusOK, tenants)
	}
}

func GetTenant(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Parse tenant ID
		tenantID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
		if err != nil || tenantID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
			return
		}

		// Get landlord ID from JWT
		userID, exists := ctx.Get("user_id")
		if !exists {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		landlordID := uint(userID.(float64))

		var tenant models.TenantWithUnit

		query := `
			SELECT
				t.id,
				t.tenant_name,
				t.payment_no1,
				t.payment_no2,
				t.rent,
				t.balance,
				t.unit_id,
				u.unit_name,
				u.unit_type,
				u.unit_price,
				t.created_at,
				t.updated_at
			FROM tenants t
			JOIN units u ON u.id = t.unit_id
			WHERE t.id = $1 AND t.landlord_id = $2
		`

		err = db.DB.QueryRow(query, tenantID, landlordID).Scan(
			&tenant.ID,
			&tenant.TenantName,
			&tenant.PaymentNo1,
			&tenant.PaymentNo2,
			&tenant.Rent,
			&tenant.Balance,
			&tenant.UnitID,
			&tenant.UnitName,
			&tenant.UnitType,
			&tenant.UnitPrice,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
		)

		if err != nil {
			if err == sql.ErrNoRows {
				ctx.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, tenant)
	}
}

func UpdateTenant(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Parse tenant ID
		tenantID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
		if err != nil || tenantID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
			return
		}

		// Get landlord ID from JWT
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

		// Partial update input
		var input struct {
			TenantName *string  `json:"tenant_name"`
			PaymentNo1 *string  `json:"payment_no1"`
			PaymentNo2 *string  `json:"payment_no2"`
			Rent       *float64 `json:"rent"`
		}

		if err := ctx.ShouldBindJSON(&input); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		query := `
			UPDATE tenants
			SET
				tenant_name = COALESCE($1, tenant_name),
				payment_no1 = COALESCE($2, payment_no1),
				payment_no2 = COALESCE($3, payment_no2),
				rent = COALESCE($4, rent),
				updated_at = NOW()
			WHERE id = $5 AND landlord_id = $6
			RETURNING
				id,
				tenant_name,
				payment_no1,
				payment_no2,
				unit_id,
				rent,
				balance,
				created_at,
				updated_at
		`

		var tenant models.Tenant

		err = db.DB.QueryRow(
			query,
			input.TenantName,
			input.PaymentNo1,
			input.PaymentNo2,
			input.Rent,
			tenantID,
			landlordID,
		).Scan(
			&tenant.ID,
			&tenant.TenantName,
			&tenant.PaymentNo1,
			&tenant.PaymentNo2,
			&tenant.UnitID,
			&tenant.Rent,
			&tenant.Balance,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
		)

		if err != nil {
			if err == sql.ErrNoRows {
				ctx.JSON(http.StatusNotFound, gin.H{
					"error": "tenant not found or not owned by user",
				})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, tenant)
	}
}

// RemoveTenant removes tenant from the database, incase he vacates
func RemoveTenant(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Parse tenant ID
		tenantID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
		if err != nil || tenantID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
			return
		}

		// Get landlord ID from JWT
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

		// Begin transaction
		tx, err := db.DB.Begin()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		committed := false
		defer func() {
			if !committed {
				tx.Rollback()
			}
		}()

		// Get unit_id & verify ownership
		var unitID uint
		err = tx.QueryRow(
			`
			SELECT unit_id
			FROM tenants
			WHERE id = $1 AND landlord_id = $2
			`,
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

		// Delete tenant
		result, err := tx.Exec(
			`DELETE FROM tenants WHERE id = $1`,
			tenantID,
		)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
			return
		}

		// Mark unit as vacant
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
		committed = true

		ctx.JSON(http.StatusOK, gin.H{
			"message": "tenant removed successfully",
		})
	}
}
