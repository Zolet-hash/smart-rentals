package main

import (
	"github.com/Zolet-hash/smart-rentals/internal/api"
	"github.com/gin-gonic/gin"
)

func main() {
	//Load configuration
	// cfg, err := config.Load()
	// if err != nil {
	// 	log.Fatal("Failed to load config:", err)
	// }

	// // Initialize database
	// db, err := database.NewDatabase(cfg.GetDSN())
	// if err != nil {
	// 	log.Fatal("Failed to connect to database:", err)
	// }
	// defer db.DB.Close()

	// // Set Gin mode
	// if cfg.Environment == "production" {
	// 	gin.SetMode(gin.ReleaseMode)
	// }

	// Initialize router with middleware
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	api.SetupRoutes(r)

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// // Initialize handlers with JWT configuration
	// authHandler := handlers.NewAuthHandler(db, []byte(cfg.JWT.Secret))

	// // Public routes
	// public := r.Group("/api/v1")
	// {
	// 	//public.POST("/register", authHandler.Register)
	// 	public.POST("/login", authHandler.Login)
	// }

	// adminRoutes := r.Group("/sudo")
	// adminRoutes.Use(
	// 	middleware.AuthMiddleware([]byte(cfg.JWT.Secret)), //sets uuid in cintext
	// 	middleware.RequireRole(db.DB, "admin"), //checks role
	// )
	// {
	// 	adminRoutes.POST("register", authHandler.Register)
	// }

	// // Protected routes with JWT middleware
	// protected := r.Group("/api/v1")
	// protected.Use(middleware.AuthMiddleware([]byte(cfg.JWT.Secret)))
	// {
	// 	protected.POST("/refresh-token", authHandler.RefreshToken)
	// 	protected.POST("/logout", authHandler.Logout)
	// 	protected.GET("/profile", getUserProfile)
	// }

	// Start server with configured host and port
	// 	serverAddr := cfg.Server.Host + ":" + cfg.Server.Port
	// 	log.Printf("Server starting on %s", serverAddr)

	// 	srv := &http.Server{
	// 		Addr:         serverAddr,
	// 		Handler:      r,
	// 		ReadTimeout:  cfg.Server.ReadTimeout,
	// 		WriteTimeout: cfg.Server.WriteTimeout,
	// 	}

	// 	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
	// 		log.Fatal("Server failed to start:", err)
	// 	}
	r.Run(":8080")
}
