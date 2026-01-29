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
	r.Use(middleware.CORS(cfg))

	authHandler := handlers.NewAuthHandler(db, []byte(cfg.JWT.Secret))

	// API v1
	api := r.Group("/api/v1")

	// Public routes
	api.POST("/login", authHandler.Login)
	api.POST("/mpesa/confirmation", handlers.MpesaPaymentConfirmation(db))

	// Admin routes
	admin := r.Group("/sudo")
	admin.Use(
		middleware.AuthMiddleware([]byte(cfg.JWT.Secret)), // sets user_id
		middleware.RequireRole(db, "admin"),               // enforces role
	)
	{
		admin.POST("/register", authHandler.Register)
	}

	// Landlord routes
	landlord := api.Group("/")
	landlord.Use(
		middleware.AuthMiddleware([]byte(cfg.JWT.Secret)),
	)
	{
		// Properties
		landlord.POST("/properties", handlers.CreateProperty(db))
		landlord.GET("/properties", handlers.ListProperties(db))
		landlord.GET("/properties/:propertyId", handlers.GetProperty(db))
		landlord.PATCH("/properties/:propertyId", handlers.UpdateProperty(db))

		// Units
		landlord.POST("/properties/:propertyId/units", handlers.CreateUnit(db))
		landlord.GET("/properties/:propertyId/units", handlers.GetUnitsByProperty(db))

		// Tenants
		landlord.POST("/units/:unitId/tenants", handlers.CreateTenant(db))
		landlord.GET("/units/:unitId/tenants", handlers.ListAllTenants(db))
		landlord.GET("/tenants/:tenantId", handlers.GetTenant(db))
		landlord.PUT("/tenants/:tenantId", handlers.UpdateTenant(db))
		landlord.DELETE("/tenants/:tenantId", handlers.RemoveTenant(db))
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
