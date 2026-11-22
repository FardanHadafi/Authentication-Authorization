package auth

import "time"

type CreateUserRequest struct {
	FullName     string `json:"full_name" validate:"required,min=2,max=255"`
	Email        string `json:"email" validate:"required,email,min=1,max=255"`
	PasswordHash string `json:"password" validate:"required,min=8,max=255"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuthResponse struct {
}
