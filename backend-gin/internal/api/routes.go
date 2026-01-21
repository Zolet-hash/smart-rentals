package api

import (
	"log"

	"github.com/Zolet-hash/smart-rentals/internal/api/handlers"
	"github.com/Zolet-hash/smart-rentals/internal/api/middleware"
	"github.com/Zolet-hash/smart-rentals/internal/database"
	"github.com/Zolet-hash/smart-rentals/internal/pkg/config"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config(routes.go)")
	}

	db, err := database.NewDatabase(cfg.GetDSN())
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	//defer db.DB.Close()

	authHandler := handlers.NewAuthHandler(db, []byte(cfg.JWT.Secret))

	// Public routes
	public := r.Group("/api/v1")
	{
		//public.POST("/register", authHandler.Register)
		public.POST("/login", authHandler.Login)
	}

	adminRoutes := r.Group("/sudo")
	adminRoutes.Use(
		middleware.AuthMiddleware([]byte(cfg.JWT.Secret)), //sets uuid in cintext
		middleware.RequireRole(db, "admin"),               //checks role
	)
	{
		adminRoutes.POST("register", authHandler.Register)
	}

	// Protected routes with JWT middleware
	protected := r.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware([]byte(cfg.JWT.Secret)))
	{
		protected.POST("/refresh-token", authHandler.RefreshToken)
		protected.POST("/logout", authHandler.Logout)
		protected.GET("/profile", getUserProfile)
	}

		// LANDLORD ROUTES
	landlord := api.Group("/landlord")
	landlord.Use(middleware.RequireRole("landlord"))
	{
		// Properties
		landlord.POST("/properties", controllers.CreateProperty)
		landlord.GET("/properties", controllers.ListProperties)
		landlord.GET("/properties/:propertyId", controllers.GetProperty)
		landlord.PUT("/properties/:propertyId", controllers.UpdateProperty) --At the moment NO [K.I.S.S]
		landlord.DELETE("/properties/:propertyId", controllers.DeleteProperty) --K.I.S.S

		// Units (nested under properties)
		landlord.POST("/properties/:propertyId/newUnit", controllers.CreateUnit)
		landlord.GET("/properties/:propertyId/units", controllers.ListUnits)
		landlord.GET("/units/:unitId", controllers.GetUnit)
		landlord.PUT("/units/:unitId", controllers.UpdateUnit) --At the moment no
		landlord.DELETE("/units/:unitId", controllers.DeleteUnit) --KISS principle

		// Tenants (assigned to units)
		landlord.POST("/units/:unitId/tenants", controllers.AddTenant)
		landlord.GET("/units/:unitId/tenants", controllers.ListTenants)
		landlord.GET("/tenants/:tenantId", controllers.GetTenant)
		landlord.PUT("/tenants/:tenantId", controllers.UpdateTenant)
		landlord.DELETE("/tenants/:tenantId", controllers.RemoveTenant)
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

// func RegisterRoutes(router *gin.Engine) {

// 	api := router.Group("/api")
// 	// PUBLIC ROUTES
// 	auth := api.Group("/auth")
// 	{
// 		auth.POST("/login", controllers.Login)
// 	}
// 	// AUTHENTICATED ROUTES
// 	api.Use(middleware.AuthMiddleware())

// 	admin := api.Group("/admin")
// 	admin.Use(middleware.RequireRole("admin"))
// 	{
// 		// User (Landlord) management --We have 2 users admin and landlord
// 		admin.POST("/users", controllers.CreateUser)           // create landlord
// 		admin.GET("/users", controllers.ListUsers)
// 		admin.GET("/users/:id", controllers.GetUser)
// 		admin.PUT("/users/:id", controllers.UpdateUser)
// 		admin.DELETE("/users/:id", controllers.DeleteUser)
// 	}
	// LANDLORD ROUTES
	landlord := api.Group("/landlord")
	landlord.Use(middleware.RequireRole("landlord"))
	{
		// Properties
		landlord.POST("/properties", controllers.CreateProperty)
		landlord.GET("/properties", controllers.ListProperties)
		landlord.GET("/properties/:propertyId", controllers.GetProperty)
		landlord.PUT("/properties/:propertyId", controllers.UpdateProperty) --At the moment NO [K.I.S.S]
		landlord.DELETE("/properties/:propertyId", controllers.DeleteProperty) --K.I.S.S

		// Units (nested under properties)
		landlord.POST("/properties/:propertyId/newUnit", controllers.CreateUnit)
		landlord.GET("/properties/:propertyId/units", controllers.ListUnits)
		landlord.GET("/units/:unitId", controllers.GetUnit)
		landlord.PUT("/units/:unitId", controllers.UpdateUnit) --At the moment no
		landlord.DELETE("/units/:unitId", controllers.DeleteUnit) --KISS principle

		// Tenants (assigned to units)
		landlord.POST("/units/:unitId/tenants", controllers.AddTenant)
		landlord.GET("/units/:unitId/tenants", controllers.ListTenants)
		landlord.GET("/tenants/:tenantId", controllers.GetTenant)
		landlord.PUT("/tenants/:tenantId", controllers.UpdateTenant)
		landlord.DELETE("/tenants/:tenantId", controllers.RemoveTenant)
	}
}
