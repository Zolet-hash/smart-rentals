package handlers

import (
	"backend-gin/internal/database"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	db        *database.Database
	jwtSecret []byte
	//Add token expiration configration
	tokenExpiration time.Duration
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(db *database.Database, jwtSecret []byte) *AuthHandler {
	return &AuthHandler{
		db:              db,
		jwtSecret:       jwtSecret,
		tokenExpiration: 24 * time.Hour,
	}
}

// Register handles User Registration
func (h *AuthHandler) Register(c *gin.Context) {
	
}
