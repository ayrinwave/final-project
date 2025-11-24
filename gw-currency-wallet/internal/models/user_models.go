package models

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// User представляет пользователя системы
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// RegisterRequest запрос на регистрацию
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
	Email    string `json:"email" validate:"required,email"`
}

// RegisterResponse ответ на регистрацию
type RegisterResponse struct {
	Message string `json:"message"`
}

// LoginRequest запрос на авторизацию
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse ответ на авторизацию
type LoginResponse struct {
	Token string `json:"token"`
}

// JWTClaims кастомные claims для JWT токена
type JWTClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	jwt.RegisteredClaims
}

func (r RegisterRequest) Validate() error {
	if r.Username == "" {
		return errors.New("username is required")
	}
	if len(r.Username) < 3 || len(r.Username) > 50 {
		return errors.New("username must be 3-50 characters")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	if len(r.Password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	if r.Email == "" {
		return errors.New("email is required")
	}
	return nil
}
