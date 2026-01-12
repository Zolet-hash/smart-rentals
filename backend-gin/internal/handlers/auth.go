package handlers

import (
	"backend-gin/internal/database"
	"backend-gin/internal/models"
	"backend-gin/internal/utils"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
		"message": "user registered successfuly",
		"user_id": id,
	})
}

// Login handlers user authentication and JWT generation
func (h *AuthHandler) Login(c *gin.Context) {
	var login models.UserLogin
	if err := c.ShouldBindJSON(&login); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid login data"}) //400
		return
	}

	//Get user from database
	var user models.User
	err := h.db.DB.QueryRow(`
	SELECT id, email password_hash
	FROM users
	WHERE email = $1`,
		login.Email,
	).Scan(user.ID, user.Email, user.PasswordHash)

	if err == sql.ErrNoRows {
		//server shouldn't give much info
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login process failed"})
		return
	}

	//Verify password
	if !utils.CheckPasswordHash(login.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	//Genarate JWT with Claims
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"iat":     now.Unix(),
		"exp":     now.Add(h.tokenExpiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tokenString, err := token.SignedString(h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}

	//Return token with expiration
	c.JSON(http.StatusOK, gin.H{
		"token_string": tokenString,
		"expires_in":   h.tokenExpiration.Seconds(),
		"token_type":   "Bearer",
	})
}

// RefreshToken generates a new token for valid users
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	//Get user ID fro context(set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	//Generate new token
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id": userID,
		"iat":     now.Unix(),
		"exp":     now.Add(h.tokenExpiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token refresh failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      tokenString,
		"expires_in": h.tokenExpiration.Seconds(),
		"token_type": "Bearer",
	})
}

// Logout endpoint (optional - useful for client-side cleanup)
func (h *AuthHandler) Logout(c *gin.Context) {
	// Since JWT is stateless, server-side logout isn't needed
	// However, we can return instructions for the client
	c.JSON(http.StatusOK, gin.H{
		"message":      "Successfully logged out",
		"instructions": "Please remove the token from your client storage",
	})
}
