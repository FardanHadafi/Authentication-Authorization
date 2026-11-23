package auth

import "time"

type CreateUserRequest struct {
	FullName     string `json:"full_name" validate:"required,min=2,max=255"`
	Email        string `json:"email" validate:"required,email,min=1,max=255"`
	PasswordHash string `json:"password" validate:"required,min=8,max=255"`
}

type LoginUserRequest struct {
	Email    string `json:"email" validate:"required,email,min=1,max=255"`
	Password string `json:"password" validate:"required,min=8,max=255"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email,min=1,max=255"`
}

type ResetPasswordRequest struct {
	Token              string `json:"token" validate:"required"`
	NewPassword        string `json:"new_password" validate:"required,min=8,max=255"`
	ConfirmNewPassword string `json:"confirm_new_password" validate:"required,eqfield=NewPassword,min=8,max=255"`
}

type ResendVerificationEmailRequest struct {
	Email string `json:"email" validate:"required,email,min=1,max=255"`
}

type ResetPasswordResponse struct {
	Message string `json:"message"`
}

type ResendVerificationEmailResponse struct {
	Message string `json:"message"`
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
	TokenType    string       `json:"token_type,omitempty"` // "Bearer"
	AccessToken  string       `json:"access_token,omitempty"`
	RefreshToken string       `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time    `json:"expires_at"`
	User         UserResponse `json:"user"`
}
