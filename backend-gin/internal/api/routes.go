package api

import (
	"github.com/Zolet-hash/smart-rentals/internal/api/handlers"
	"github.com/Zolet-hash/smart-rentals/internal/api/middleware"
	"github.com/Zolet-hash/smart-rentals/internal/database"
	"github.com/Zolet-hash/smart-rentals/internal/pkg/config"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(
	r *gin.Engine,
	db *database.Database,
	cfg *config.Config,
) {
	// Global middleware
	r.Use(middleware.CORS())

	authHandler := handlers.NewAuthHandler(db, []byte(cfg.JWT.Secret))

	// API v1
	api := r.Group("/api/v1")

	// Public routes
	api.POST("/login",
		middleware.RateLimiter(), // <- limit requests
		authHandler.Login,
	)
	api.POST("/validation", handlers.MpesaValidation)
	api.POST("/confirmation", handlers.MpesaPaymentConfirmation(db))
	api.POST("/register", authHandler.Register)

	// Admin routes
	admin := r.Group("/sudo")
	admin.Use(
		middleware.AuthMiddleware([]byte(cfg.JWT.Secret)), // secret from env

	)
	{
		//admin.POST("/register", authHandler.Register)
	}

	// Landlord routes
	landlord := api.Group("/")
	landlord.Use(
		middleware.AuthMiddleware([]byte(cfg.JWT.Secret)),
	)
	{
		// Properties
		landlord.POST("/properties", handlers.CreateProperty(db))
		landlord.POST("/paybill", handlers.AttachPaybill(db))
		landlord.POST("/till", handlers.AttachTill(db))
		landlord.GET("/properties", handlers.ListProperties(db))
		landlord.GET("/properties/:propertyId", handlers.GetProperty(db))
		landlord.PATCH("/properties/:propertyId", handlers.UpdateProperty(db))

		// Units
		landlord.POST("/properties/:propertyId/units", handlers.CreateUnit(db))
		landlord.GET("/properties/:propertyId/units", handlers.GetUnitsByProperty(db))

		// Tenants
		landlord.POST("/units/:unitId/tenants", handlers.CreateTenant(db))
		landlord.GET("/properties/:propertyId/tenants", handlers.ListPropertyTenants(db))
		landlord.GET("/properties/:propertyId/tenants/:tenantId", handlers.GetPropertyTenant(db))
		landlord.PATCH("/properties/:propertyId/tenants/:tenantId", handlers.UpdatePropertyTenant(db))
		landlord.DELETE("/properties/:propertyId/tenants/:tenantId", handlers.RemovePropertyTenant(db))

		// Payments
		landlord.GET("/properties/:propertyId/payments")
		landlord.POST("/confirmation-message", handlers.RecordSendMoney(db))

		landlord.GET("/billing", handlers.GetLandlordBilling(db))
	}
}

func getUserProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	email, _ := c.Get("email")

	c.JSON(200, gin.H{
		"user_id": userID,
		"email":   email,
	})
}
