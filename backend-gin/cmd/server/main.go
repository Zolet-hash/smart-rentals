package main

import (
	"github.com/Zolet-hash/smart-rentals/internal/api"
	"github.com/gin-gonic/gin"
)

func main() {

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

	r.Run(":8080")
}
