package handlers

import (
	"backend-gin/internal/database"
	"backend-gin/internal/models"
	"backend-gin/internal/utils"
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

	//Hash password
	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Password processing failed"})
		return
	}

	//Insert user with transaction
	tx, err := h.db.DB.Begin() //group multiple database operations
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction start failed"})
		return
	}
	var id int
	err = tx.QueryRow(`
		INSERT INTO users (email, password_hash)
		VALUES ($1 $2)
		RETURNING id`,
		user.Email, hashedPassword,
	).Scan(&id)

	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User creation failed"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "user registeres successfuly",
		"user_id": id,
	})

}
