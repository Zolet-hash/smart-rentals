package middleware

import (
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS returns a Gin middleware configured for Vercel + local dev + previews
func CORS() gin.HandlerFunc {
	frontendURL := os.Getenv("FRONTEND_URL") // e.g., https://my-app.vercel.app

	return cors.New(cors.Config{
		AllowOrigins: []string{
			frontendURL,             // production
			"http://localhost:3000", // local dev
		},
		// Support Vercel preview deployments
		AllowOriginFunc: func(origin string) bool {
			if origin == "" {
				return false
			}
			if origin == frontendURL || origin == "http://localhost:3000" {
				return true
			}
			// Allow any *.vercel.app preview
			return strings.HasSuffix(origin, ".vercel.app")
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
