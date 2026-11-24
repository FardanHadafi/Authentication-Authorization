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
	ID           string    `json:"id"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	TokenVersion int       `json:"token_version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AuthResponse struct {
	TokenType    string       `json:"token_type,omitempty"` // "Bearer"
	AccessToken  string       `json:"access_token,omitempty"`
	RefreshToken string       `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time    `json:"expires_at"`
	User         UserResponse `json:"user"`
}

type AuditLogResponse struct {
	ID           string         `json:"id"`
	ActorID      *string        `json:"actor_id,omitempty"`
	ActorEmail   *string        `json:"actor_email,omitempty"`
	ActorRole    *string        `json:"actor_role,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   *string        `json:"resource_id,omitempty"`
	Status       string         `json:"status"`
	IPAddress    string         `json:"ip_address"`
	UserAgent    string         `json:"user_agent"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type PaginatedAuditLogsResponse struct {
	Data       []AuditLogResponse `json:"data"`
	TotalCount int64              `json:"total_count"`
	Page       int32              `json:"page"`
	Limit      int32              `json:"limit"`
}

type CSRFTokenResponse struct {
	CSRFToken string `json:"csrf_token"`
}

type UserPermissionsResponse struct {
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}
