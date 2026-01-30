package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/Zolet-hash/smart-rentals/internal/database"
	"github.com/Zolet-hash/smart-rentals/internal/models"
	"github.com/gin-gonic/gin"
)

//CreateTenant creates a new tenant in a property
func CreateTenant(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var tenant models.Tenant

		// Bind JSON body (tenant_name, payment_no1, payment_no2, rent)
		if err := ctx.ShouldBindJSON(&tenant); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tenant.PaymentNo1 = normalizePhone(tenant.PaymentNo1)
		tenant.PaymentNo2 = normalizePhone(tenant.PaymentNo2)

		if tenant.Rent <= 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "rent must be greater than zero"})
			return
		}

		// Get unit ID from URL
		unitIdParam := ctx.Param("unitId")
		unitID, err := strconv.ParseUint(unitIdParam, 10, 64)
		if err != nil || unitID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
			return
		}
		tenant.UnitID = uint(unitID)

		// Get landlord ID from JWT (HttpOnly cookie)
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

		err = tx.QueryRow(ownershipQuery, tenant.UnitID, landlordID).Scan(&isVacant)
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

		// Insert tenant
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
		).Scan(&tenant.ID, &tenant.Balance, &tenant.CreatedAt, &tenant.UpdatedAt)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Mark unit as occupied
		_, err = tx.Exec(`
			UPDATE units
			SET vacancy = false,
			    updated_at = NOW()
			WHERE id = $1
		`, tenant.UnitID)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		//Commit transaction
		if err := tx.Commit(); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		committed = true

		tenant.LandlordID = landlordID

		//Return created tenant
		ctx.JSON(http.StatusCreated, tenant)
	}
}

//ListPropertyTenants lists all tenants in a specific property
func ListPropertyTenants(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Get landlord ID from JWT (HttpOnly cookie)
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

		// Get property ID from URL
		propertyIdParam := ctx.Param("propertyId")
		propertyID, err := strconv.ParseUint(propertyIdParam, 10, 64)
		if err != nil || propertyID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid property id"})
			return
		}

		// Verify landlord owns the property
		var ownsProperty bool
		err = db.DB.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM properties
				WHERE id = $1 AND landlord_id = $2
			)
		`, propertyID, landlordID).Scan(&ownsProperty)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if !ownsProperty {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "you do not own this property"})
			return
		}

		// Fetch tenants under this property
		query := `
			SELECT
				t.id,
				t.tenant_name,
				t.payment_no1,
				t.payment_no2,
				t.rent,
				t.balance,
				u.id AS unit_id,
				u.unit_name,
				u.unit_type,
				u.unit_price,
				t.created_at,
				t.updated_at
			FROM tenants t
			JOIN units u ON u.id = t.unit_id
			WHERE u.property_id = $1
			  AND t.landlord_id = $2
			ORDER BY t.created_at DESC
		`

		rows, err := db.DB.Query(query, propertyID, landlordID)
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

//GetPropertyTenant gets a specific tenant on a property
func GetPropertyTenant(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Parse property ID from URL
		propertyIdParam := ctx.Param("propertyId")
		propertyID, err := strconv.ParseUint(propertyIdParam, 10, 64)
		if err != nil || propertyID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid property id"})
			return
		}

		//  Parse tenant ID from URL
		tenantIdParam := ctx.Param("tenantId")
		tenantID, err := strconv.ParseUint(tenantIdParam, 10, 64)
		if err != nil || tenantID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
			return
		}

		// Get landlord ID from JWT (HttpOnly cookie)
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

		var tenant models.TenantWithUnit

		// Fetch tenant scoped to property ownership
		query := `
			SELECT
				t.id,
				t.tenant_name,
				t.payment_no1,
				t.payment_no2,
				t.rent,
				t.balance,
				u.id AS unit_id,
				u.unit_name,
				u.unit_type,
				u.unit_price,
				t.created_at,
				t.updated_at
			FROM tenants t
			JOIN units u ON u.id = t.unit_id
			JOIN properties p ON p.id = u.property_id
			WHERE t.id = $1
			  AND p.id = $2
			  AND p.landlord_id = $3
		`

		err = db.DB.QueryRow(query, tenantID, propertyID, landlordID).Scan(
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
				ctx.JSON(http.StatusNotFound, gin.H{
					"error": "tenant not found in this property",
				})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, tenant)
	}
}

func UpdatePropertyTenant(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Parse property ID
		propertyIdParam := ctx.Param("propertyId")
		propertyID, err := strconv.ParseUint(propertyIdParam, 10, 64)
		if err != nil || propertyID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid property id"})
			return
		}

		// Parse tenant ID
		tenantIdParam := ctx.Param("tenantId")
		tenantID, err := strconv.ParseUint(tenantIdParam, 10, 64)
		if err != nil || tenantID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
			return
		}

		// Get landlord ID from JWT (HttpOnly cookie)
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

		// Optional validation
		if input.Rent != nil && *input.Rent <= 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "rent must be greater than zero"})
			return
		}

		// Update tenant scoped to property ownership
		query := `
			UPDATE tenants t
			SET
				tenant_name = COALESCE($1, t.tenant_name),
				payment_no1 = COALESCE($2, t.payment_no1),
				payment_no2 = COALESCE($3, t.payment_no2),
				rent        = COALESCE($4, t.rent),
				updated_at  = NOW()
			FROM units u
			JOIN properties p ON p.id = u.property_id
			WHERE t.id = $5
			  AND t.unit_id = u.id
			  AND p.id = $6
			  AND p.landlord_id = $7
			RETURNING
				t.id,
				t.tenant_name,
				t.payment_no1,
				t.payment_no2,
				t.unit_id,
				t.rent,
				t.balance,
				t.created_at,
				t.updated_at
		`

		var tenant models.Tenant

		err = db.DB.QueryRow(
			query,
			input.TenantName,
			input.PaymentNo1,
			input.PaymentNo2,
			input.Rent,
			tenantID,
			propertyID,
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
					"error": "tenant not found in this property",
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
func RemovePropertyTenant(db *database.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 1️⃣ Parse property ID
		propertyIdParam := ctx.Param("propertyId")
		propertyID, err := strconv.ParseUint(propertyIdParam, 10, 64)
		if err != nil || propertyID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid property id"})
			return
		}

		// Parse tenant ID
		tenantIdParam := ctx.Param("tenantId")
		tenantID, err := strconv.ParseUint(tenantIdParam, 10, 64)
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

		// Delete tenant scoped to property ownership
		var deletedTenantID uint
		query := `
			DELETE FROM tenants t
			USING units u
			JOIN properties p ON p.id = u.property_id
			WHERE t.id = $1
			  AND t.unit_id = u.id
			  AND p.id = $2
			  AND p.landlord_id = $3
			RETURNING t.id
		`
		err = db.DB.QueryRow(query, tenantID, propertyID, landlordID).Scan(&deletedTenantID)
		if err != nil {
			if err == sql.ErrNoRows {
				ctx.JSON(http.StatusNotFound, gin.H{
					"error": "tenant not found in this property",
				})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Mark unit as vacant
		_, err = db.DB.Exec(`
			UPDATE units
			SET vacancy = true,
			    updated_at = NOW()
			WHERE id = (SELECT unit_id FROM tenants WHERE id = $1)
		`, tenantID)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"message": "tenant removed successfully",
		})
	}
}
