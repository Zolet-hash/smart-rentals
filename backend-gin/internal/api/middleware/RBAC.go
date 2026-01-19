package middleware

import "github.com/gin-gonic/gin"

func AdminMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		role, exists := ctx.Get("role")
		if !exists {
			ctx.JSON(500, gin.H{"error": "Role not found in context"})
			ctx.Abort()
			return
		}
		if role != "admin" {
			ctx.JSON(403, gin.H{"error": "Admin access required"})
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

func LandlordMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		role, exists := ctx.Get("role")
		if !exists {
			ctx.JSON(500, gin.H{"error": "Role not found in context"})
			ctx.Abort()
			return
		}
		if role != "landlord" {
			ctx.JSON(403, gin.H{"error": "Landlord access required"})
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

func TenantMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		role, exists := ctx.Get("role")
		if !exists {
			ctx.JSON(500, gin.H{"error": "Role not found in context"})
			ctx.Abort()
			return
		}
		if role != "tenant" {
			ctx.JSON(403, gin.H{"error": "Tenant access required"})
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
