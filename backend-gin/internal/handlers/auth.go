package handlers

import (
	"backend-gin/internal/database"
	"backend-gin/internal/models"
	"net/http"
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
	var user models.UserRegister

	//validate input json
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid input format",
			"details": err.Error(),
		})
		return
	}

	//Validations to check if email is valid
	if err := user.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	//check if user already exists
	var exists bool
	err := h.db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", user.Email).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

}
