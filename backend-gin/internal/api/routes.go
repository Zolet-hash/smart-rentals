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
		public.POST("/login", authHandler.Login)
	}

	adminRoutes := r.Group("/sudo")
	adminRoutes.Use(
		middleware.AuthMiddleware([]byte(cfg.JWT.Secret)), //sets uuid in context
		middleware.RequireRole(db, "admin"),               //checks role
	)
	{
		adminRoutes.POST("register", authHandler.Register) //register landlord
		//admin.GET("/users", controllers.ListLandlords) --no at the moment
 		//admin.GET("/users/:id", controllers.GetLandlord) --since we only have 1 landlord
 		//admin.PATCH("/users/:id", controllers.Updatelandlord)
		//admin.DELETE("/users/:id", controllers.Deletelandlord)
	}


	// LANDLORD ROUTES
	landlord := r.Group("/api/v1")
	landlord.Use(middleware.AuthMiddleware([]byte(cfg.JWT.Secret)))
	{
		// Properties
		landlord.POST("/properties/newProperty", handlers.CreateProperty(db))
		landlord.GET("/properties/myProperties", handlers.ListProperties(db))
		landlord.GET("/properties/:propertyId", handlers.GetProperty(db))
		landlord.PATCH("/properties/:updatePropertyByID", handlers.UpdateProperty(db))
		//landlord.DELETE("/properties/:propertyId", controllers.DeleteProperty) //--K.I.S.S

		// Units (nested under properties)
		landlord.POST("/properties/:propertyId/newUnit", handlers.CreateUnit(db))
		landlord.GET("/properties/:propertyId/units", handlers.GetUnitsByProperty(db))
		//landlord.GET("/units/:unitId", controllers.GetUnit) --jj
	    //landlord.PUT("/units/:unitId", controllers.UpdateUnit)
		//landlord.DELETE("/units/:unitId", controllers.DeleteUnit) --ii

		// Tenants (assigned to units)
		landlord.POST("/units/:unitId/addTenant", handlers.CreateTenant(db))
		landlord.GET("/units/:unitId/myTenants", handlers.ListAllTenants(db))
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

