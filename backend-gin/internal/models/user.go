package models

import (
	"errors"
	"regexp"
	"time"
)

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
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// UserRegister represents registration reqest data
type UserRegister struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// Validate checks if email format is valid
func (u *UserRegister) Validate() error {
	emailRegexp := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	if !emailRegexp.MatchString(u.Email) {
		return errors.New("invalid email format")
	}
	return nil
}
