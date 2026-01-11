package models

import "time"

// User represents our database user
type User struct {
	ID           int
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` //"-" means this wont be included in JSON
	CreatedAT    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserLogin represents new login request
type UserLogin struct {
	Email    string `json:"email" binding:"reqiured,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// UserRegister represents registration reqest data
type UserRegister struct {
	Email    string `json:"email" binding:"reqiured,email"`
	Password string `json:"password" binding:"required,min=6"`
}
