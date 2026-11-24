package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"authentication-authorization/shared"
	"authentication-authorization/shared/db"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

var (
	ErrEmailAlreadyExists        = errors.New("email already registered")
	ErrInvalidOrExpiredToken     = errors.New("invalid or expired verification token")
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrAccountNotVerified        = errors.New("email not verified, please check your email inbox")
	ErrRedisClientNotInitialized = errors.New("redis client not initialized")
	ErrUserNotFound              = errors.New("user not found")
	ErrAccountInactive           = errors.New("account inactive")
)

type AuthService interface {
	Register(ctx context.Context, request CreateUserRequest) (UserResponse, error)
	VerifyEmail(ctx context.Context, token string) error
	Login(ctx context.Context, request LoginUserRequest) (AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (AuthResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	ForgotPassword(ctx context.Context, request ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, request ResetPasswordRequest) error
	ResendVerificationEmail(ctx context.Context, request ResendVerificationEmailRequest) error
	GetProfile(ctx context.Context, userID uuid.UUID) (UserResponse, error)
	ListUsers(ctx context.Context) ([]UserResponse, error)
	ListAuditLogs(ctx context.Context, page, limit int32) (PaginatedAuditLogsResponse, error)
}

type authService struct {
	querier      db.Querier
	asynqClient  *asynq.Client
	mailer       shared.EmailSender
	redis        *redis.Client
	tokenManager shared.TokenManager
	auditLogger  shared.AuditLogger
}

func NewAuthService(querier db.Querier, asynqClient *asynq.Client, mailer shared.EmailSender, redis *redis.Client, tokenManager shared.TokenManager, auditLogger shared.AuditLogger) AuthService {
	return &authService{
		querier:      querier,
		asynqClient:  asynqClient,
		mailer:       mailer,
		redis:        redis,
		tokenManager: tokenManager,
		auditLogger:  auditLogger,
	}
}

func (a *authService) Register(ctx context.Context, request CreateUserRequest) (UserResponse, error) {
	// 1. Sanitasi email user
	request.Email = strings.TrimSpace(strings.ToLower(request.Email))

	// 2. Cek apakah email sudah terdaftar
	existingUser, err := a.querier.GetUserByEmail(ctx, request.Email)
	if err == nil && existingUser.ID != uuid.Nil {
		return UserResponse{}, ErrEmailAlreadyExists
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return UserResponse{}, fmt.Errorf("failed to check existing user: %w", err)
	}

	// 3. Hash password
	hashedPassword, err := argon2id.CreateHash(request.PasswordHash, argon2id.DefaultParams)
	if err != nil {
		return UserResponse{}, fmt.Errorf("failed to hash password: %w", err)
	}

	// 4. Simpan ke database
	newUser, err := a.querier.CreateUser(ctx, db.CreateUserParams{
		FullName:     request.FullName,
		Email:        request.Email,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		return UserResponse{}, fmt.Errorf("failed to create new user: %w", err)
	}

	// 5. Generate token verifikasi, simpan ke Redis (TTL 24 jam), & kirim email via Asynq
	verificationToken := uuid.New().String()
	if a.redis != nil {
		_ = a.redis.Set(ctx, fmt.Sprintf("verify:%s", verificationToken), newUser.ID.String(), 24*time.Hour).Err()
	}

	appURL := strings.TrimRight(os.Getenv("APP_URL"), "/")
	if appURL == "" {
		appURL = "http://localhost:8080"
	}
	verificationLink := fmt.Sprintf("%s/api/auth/verify?token=%s", appURL, verificationToken)

	task, taskErr := shared.NewVerificationEmailTask(newUser.Email, newUser.FullName, verificationLink)
	if taskErr != nil {
		return UserResponse{}, fmt.Errorf("failed to create verification email task: %w", taskErr)
	}
	if a.asynqClient != nil {
		if _, err := a.asynqClient.EnqueueContext(ctx, task); err != nil {
			return UserResponse{}, fmt.Errorf("failed to enqueue verification email task: %w", err)
		}
	}

	if a.auditLogger != nil {
		a.auditLogger.Log(ctx, shared.AuditEntry{
			ActorID:      &newUser.ID,
			ActorEmail:   newUser.Email,
			ActorRole:    newUser.Role,
			Action:       "auth.register",
			ResourceType: "user",
			ResourceID:   newUser.ID.String(),
			Status:       "success",
		})
	}

	// 6. Return response yang sudah dipetakan dengan DTO
	return UserResponse{
		ID:           newUser.ID.String(),
		FullName:     newUser.FullName,
		Email:        newUser.Email,
		Role:         newUser.Role,
		Status:       newUser.Status,
		TokenVersion: int(newUser.TokenVersion),
		CreatedAt:    newUser.CreatedAt,
		UpdatedAt:    newUser.UpdatedAt,
	}, nil
}

func (a *authService) VerifyEmail(ctx context.Context, token string) error {
	if a.redis == nil {
		return ErrRedisClientNotInitialized
	}

	key := fmt.Sprintf("verify:%s", token)
	userIDStr, err := a.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrInvalidOrExpiredToken
		}
		return fmt.Errorf("failed to get verification token: %w", err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("invalid user id in token: %w", err)
	}

	// Update status user di database menjadi 'active'
	err = a.querier.UpdateUserStatus(ctx, db.UpdateUserStatusParams{
		ID:     userID,
		Status: "active",
	})
	if err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}

	// Hapus token dari Redis agar tidak bisa digunakan ulang
	_ = a.redis.Del(ctx, key).Err()

	if a.auditLogger != nil {
		a.auditLogger.Log(ctx, shared.AuditEntry{
			ActorID:      &userID,
			Action:       "auth.email_verified",
			ResourceType: "user",
			ResourceID:   userID.String(),
			Status:       "success",
		})
	}

	return nil
}

func (a *authService) Login(ctx context.Context, request LoginUserRequest) (AuthResponse, error) {
	if a.redis == nil {
		return AuthResponse{}, ErrRedisClientNotInitialized
	}

	// Sanitasi email
	request.Email = strings.TrimSpace(strings.ToLower(request.Email))

	// Ambil user dari database
	user, err := a.querier.GetUserByEmail(ctx, request.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if a.auditLogger != nil {
				a.auditLogger.Log(ctx, shared.AuditEntry{
					ActorEmail:   request.Email,
					Action:       "auth.login",
					ResourceType: "session",
					Status:       "failure",
					Metadata:     map[string]any{"reason": "user_not_found"},
				})
			}
			return AuthResponse{}, ErrInvalidCredentials
		}
		return AuthResponse{}, fmt.Errorf("failed to get user from database: %w", err)
	}

	// Cocokan password dari user dengan database
	match, err := argon2id.ComparePasswordAndHash(request.Password, user.PasswordHash)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("failed to verify password hash: %w", err)
	}

	// Jika format valid tapi hash tidak cocok
	if !match {
		if a.auditLogger != nil {
			a.auditLogger.Log(ctx, shared.AuditEntry{
				ActorID:      &user.ID,
				ActorEmail:   user.Email,
				ActorRole:    user.Role,
				Action:       "auth.login",
				ResourceType: "session",
				Status:       "failure",
				Metadata:     map[string]any{"reason": "invalid_credentials"},
			})
		}
		return AuthResponse{}, ErrInvalidCredentials
	}

	// Pastikan status akun sudah aktif
	if user.Status == "pending" {
		if a.auditLogger != nil {
			a.auditLogger.Log(ctx, shared.AuditEntry{
				ActorID:      &user.ID,
				ActorEmail:   user.Email,
				ActorRole:    user.Role,
				Action:       "auth.login",
				ResourceType: "session",
				Status:       "failure",
				Metadata:     map[string]any{"reason": "pending_verification"},
			})
		}
		return AuthResponse{}, ErrAccountNotVerified
	} else if user.Status != "active" {
		if a.auditLogger != nil {
			a.auditLogger.Log(ctx, shared.AuditEntry{
				ActorID:      &user.ID,
				ActorEmail:   user.Email,
				ActorRole:    user.Role,
				Action:       "auth.login",
				ResourceType: "session",
				Status:       "failure",
				Metadata:     map[string]any{"reason": "account_inactive"},
			})
		}
		return AuthResponse{}, errors.New("account is not active yet, please check your verification link in your mailbox")
	}

	// Terbitkan AccessToken
	accessToken, expiresAt, err := a.tokenManager.GenerateAccessToken(user.ID, user.Email, user.Role, int(user.TokenVersion))
	if err != nil {
		return AuthResponse{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Terbitkan RefreshToken
	refreshToken, err := a.tokenManager.GenerateRefreshToken()
	if err != nil {
		return AuthResponse{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Simpan refresh token ke redis dengan TTL 7 hari
	refreshKey := fmt.Sprintf("refresh:%s", refreshToken)
	if err := a.redis.Set(ctx, refreshKey, user.ID.String(), 7*24*time.Hour).Err(); err != nil {
		return AuthResponse{}, fmt.Errorf("failed to save refresh token to redis: %w", err)
	}

	if a.auditLogger != nil {
		a.auditLogger.Log(ctx, shared.AuditEntry{
			ActorID:      &user.ID,
			ActorEmail:   user.Email,
			ActorRole:    user.Role,
			Action:       "auth.login",
			ResourceType: "session",
			ResourceID:   user.ID.String(),
			Status:       "success",
		})
	}

	// Return AuthResponse
	return AuthResponse{
		TokenType:    "Bearer",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User: UserResponse{
			ID:           user.ID.String(),
			FullName:     user.FullName,
			Email:        user.Email,
			Role:         user.Role,
			Status:       user.Status,
			TokenVersion: int(user.TokenVersion),
			CreatedAt:    user.CreatedAt,
			UpdatedAt:    user.UpdatedAt,
		},
	}, nil
}

func (a *authService) RefreshToken(ctx context.Context, refreshToken string) (AuthResponse, error) {
	if a.redis == nil {
		return AuthResponse{}, ErrRedisClientNotInitialized
	}

	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return AuthResponse{}, ErrInvalidOrExpiredToken
	}

	// 1. Ambil user ID dari Redis
	key := fmt.Sprintf("refresh:%s", refreshToken)
	userIDStr, err := a.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return AuthResponse{}, ErrInvalidOrExpiredToken
		}
		return AuthResponse{}, fmt.Errorf("failed to get refresh token: %w", err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("invalid user id in refresh token: %w", err)
	}

	// 2. Refresh Token Rotation (RTR): Hapus token lama segera agar tidak bisa di-reuse
	_ = a.redis.Del(ctx, key).Err()

	// 3. Ambil data user dari database
	user, err := a.querier.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuthResponse{}, ErrInvalidOrExpiredToken
		}
		return AuthResponse{}, fmt.Errorf("failed to get user: %w", err)
	}

	// 4. Pastikan akun user tetap berstatus active
	if user.Status != "active" {
		return AuthResponse{}, errors.New("account is not active")
	}

	// 5. Generate Access Token baru
	accessToken, expiresAt, err := a.tokenManager.GenerateAccessToken(user.ID, user.Email, user.Role, int(user.TokenVersion))
	if err != nil {
		return AuthResponse{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	// 6. Generate Refresh Token baru
	newRefreshToken, err := a.tokenManager.GenerateRefreshToken()
	if err != nil {
		return AuthResponse{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// 7. Simpan refresh token baru ke Redis dengan TTL 7 hari
	newKey := fmt.Sprintf("refresh:%s", newRefreshToken)
	if err := a.redis.Set(ctx, newKey, user.ID.String(), 7*24*time.Hour).Err(); err != nil {
		return AuthResponse{}, fmt.Errorf("failed to save refresh token to redis: %w", err)
	}

	return AuthResponse{
		TokenType:    "Bearer",
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
		User: UserResponse{
			ID:           user.ID.String(),
			FullName:     user.FullName,
			Email:        user.Email,
			Role:         user.Role,
			Status:       user.Status,
			TokenVersion: int(user.TokenVersion),
			CreatedAt:    user.CreatedAt,
			UpdatedAt:    user.UpdatedAt,
		},
	}, nil
}

func (a *authService) Logout(ctx context.Context, refreshToken string) error {
	if a.redis == nil {
		return ErrRedisClientNotInitialized
	}

	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken != "" {
		key := fmt.Sprintf("refresh:%s", refreshToken)
		_ = a.redis.Del(ctx, key).Err()
	}

	if a.auditLogger != nil {
		a.auditLogger.Log(ctx, shared.AuditEntry{
			Action:       "auth.logout",
			ResourceType: "session",
			Status:       "success",
		})
	}

	return nil
}

func (a *authService) ForgotPassword(ctx context.Context, request ForgotPasswordRequest) error {
	if a.redis == nil {
		return ErrRedisClientNotInitialized
	}

	request.Email = strings.TrimSpace(strings.ToLower(request.Email))

	user, err := a.querier.GetUserByEmail(ctx, request.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Anti-user enumeration: jangan bocorkan bahwa email tidak terdaftar
			return nil
		}
		return fmt.Errorf("failed to get user from database: %w", err)
	}

	// Jangan kirim reset password jika akun belum aktif
	if user.Status != "active" {
		return nil
	}

	// Generate reset token (UUID) dan simpan ke Redis dengan TTL 15 menit
	resetToken := uuid.New().String()
	resetKey := fmt.Sprintf("reset:%s", resetToken)
	if err := a.redis.Set(ctx, resetKey, user.ID.String(), 15*time.Minute).Err(); err != nil {
		return fmt.Errorf("failed to save reset token to redis: %w", err)
	}

	// Susun URL reset password menuju frontend SvelteKit
	frontendURL := strings.TrimRight(os.Getenv("FRONTEND_URL"), "/")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, resetToken)

	// Buat task asynq dan masukkan ke antrean
	task, taskErr := shared.NewResetPasswordEmailTask(user.Email, user.FullName, resetLink)
	if taskErr != nil {
		return fmt.Errorf("failed to create reset email task: %w", taskErr)
	}
	if a.asynqClient != nil {
		if _, err := a.asynqClient.EnqueueContext(ctx, task); err != nil {
			return fmt.Errorf("failed to enqueue reset email task: %w", err)
		}
	}

	if a.auditLogger != nil {
		a.auditLogger.Log(ctx, shared.AuditEntry{
			ActorID:      &user.ID,
			ActorEmail:   user.Email,
			ActorRole:    user.Role,
			Action:       "auth.forgot_password_requested",
			ResourceType: "user",
			ResourceID:   user.ID.String(),
			Status:       "success",
		})
	}

	return nil
}

func (a *authService) ResetPassword(ctx context.Context, request ResetPasswordRequest) error {
	if a.redis == nil {
		return ErrRedisClientNotInitialized
	}

	token := strings.TrimSpace(request.Token)
	if token == "" {
		return ErrInvalidOrExpiredToken
	}

	// 1. Ambil user ID dari Redis berdasarkan token reset
	resetKey := fmt.Sprintf("reset:%s", token)
	userIDStr, err := a.redis.Get(ctx, resetKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrInvalidOrExpiredToken
		}
		return fmt.Errorf("failed to get reset token from redis: %w", err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("invalid user id in reset token: %w", err)
	}

	// 2. Segera hapus token dari Redis (One-Time Token Enforcement)
	_ = a.redis.Del(ctx, resetKey).Err()

	// 3. Pastikan user ada di database dan statusnya masih aktif
	user, err := a.querier.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidOrExpiredToken
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user.Status != "active" {
		return errors.New("account is not active")
	}

	// 4. Hash password baru menggunakan Argon2id
	hashedPassword, err := argon2id.CreateHash(request.NewPassword, argon2id.DefaultParams)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// 5. Update password di database PostgreSQL
	err = a.querier.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           user.ID,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}

	// 6. Cabut seluruh sesi aktif lama: bump token_version di PostgreSQL & Redis agar Access Token lama batal seketika!
	newVersion, err := a.querier.IncrementUserTokenVersion(ctx, user.ID)
	if err == nil {
		versionKey := fmt.Sprintf("user:%s:token_version", user.ID.String())
		_ = a.redis.Set(ctx, versionKey, strconv.Itoa(int(newVersion)), 30*24*time.Hour).Err()
	}

	// Hapus seluruh refresh token user di Redis
	iter := a.redis.Scan(ctx, 0, "refresh:*", 0).Iterator()
	for iter.Next(ctx) {
		k := iter.Val()
		val, err := a.redis.Get(ctx, k).Result()
		if err == nil && val == user.ID.String() {
			_ = a.redis.Del(ctx, k).Err()
		}
	}

	if a.auditLogger != nil {
		a.auditLogger.Log(ctx, shared.AuditEntry{
			ActorID:      &user.ID,
			ActorEmail:   user.Email,
			ActorRole:    user.Role,
			Action:       "auth.password_reset",
			ResourceType: "user",
			ResourceID:   user.ID.String(),
			Status:       "success",
			Metadata: map[string]any{
				"new_token_version": newVersion,
			},
		})
	}

	return nil
}

func (a *authService) ResendVerificationEmail(ctx context.Context, request ResendVerificationEmailRequest) error {
	if a.redis == nil {
		return ErrRedisClientNotInitialized
	}

	request.Email = strings.TrimSpace(strings.ToLower(request.Email))

	user, err := a.querier.GetUserByEmail(ctx, request.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Anti-user enumeration: jangan bocorkan bahwa email tidak terdaftar
			return nil
		}

		return fmt.Errorf("failed to get user from database: %w", err)
	}

	// Hanya kirim verifikasi ulang jika status akun memang masih pending
	if user.Status != "pending" {
		return nil
	}

	// 1. Generate token verifikasi baru (UUID)
	verificationToken := uuid.New().String()

	// 2. Simpan token verifikasi ke Redis dengan TTL 24 jam
	verifyKey := fmt.Sprintf("verify:%s", verificationToken)
	if err := a.redis.Set(ctx, verifyKey, user.ID.String(), 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to save verification token to redis: %w", err)
	}

	// 3. Susun verification link mengarah ke endpoint backend verify
	appURL := strings.TrimRight(os.Getenv("APP_URL"), "/")
	if appURL == "" {
		appURL = "http://localhost:8080"
	}
	verificationLink := fmt.Sprintf("%s/api/auth/verify?token=%s", appURL, verificationToken)

	// 4. Buat task Asynq dan masukkan ke antrean pengiriman email
	task, taskErr := shared.NewResendVerificationEmailTask(user.Email, user.FullName, verificationLink)
	if taskErr != nil {
		return fmt.Errorf("failed to create resend verification email task: %w", taskErr)
	}
	if a.asynqClient != nil {
		if _, err := a.asynqClient.EnqueueContext(ctx, task); err != nil {
			return fmt.Errorf("failed to enqueue resend verification email task: %w", err)
		}
	}

	return nil
}

func (a *authService) GetProfile(ctx context.Context, userID uuid.UUID) (UserResponse, error) {
	if userID == uuid.Nil {
		return UserResponse{}, ErrUserNotFound
	}

	user, err := a.querier.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserResponse{}, ErrUserNotFound
		}

		return UserResponse{}, fmt.Errorf("failed to get user from database: %w", err)
	}

	if user.Status != "active" {
		return UserResponse{}, ErrAccountInactive
	}

	return UserResponse{
		ID:           user.ID.String(),
		FullName:     user.FullName,
		Email:        user.Email,
		Role:         user.Role,
		Status:       user.Status,
		TokenVersion: int(user.TokenVersion),
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}, nil
}

func (a *authService) ListUsers(ctx context.Context) ([]UserResponse, error) {
	users, err := a.querier.ListUsers(ctx)
	if err != nil {
		return []UserResponse{}, fmt.Errorf("failed to get users from database: %w", err)
	}

	response := make([]UserResponse, len(users))
	for i, u := range users {
		response[i] = UserResponse{
			ID:           u.ID.String(),
			FullName:     u.FullName,
			Email:        u.Email,
			Role:         u.Role,
			Status:       u.Status,
			TokenVersion: int(u.TokenVersion),
			CreatedAt:    u.CreatedAt,
			UpdatedAt:    u.UpdatedAt,
		}
	}

	return response, nil
}

func (a *authService) ListAuditLogs(ctx context.Context, page, limit int32) (PaginatedAuditLogsResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	totalCount, err := a.querier.CountAuditLogs(ctx)
	if err != nil {
		return PaginatedAuditLogsResponse{}, fmt.Errorf("failed to count audit logs: %w", err)
	}

	logs, err := a.querier.ListAuditLogs(ctx, db.ListAuditLogsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return PaginatedAuditLogsResponse{}, fmt.Errorf("failed to list audit logs: %w", err)
	}

	data := make([]AuditLogResponse, len(logs))
	for i, l := range logs {
		var actorID *string
		if l.ActorID.Valid {
			idStr := uuid.UUID(l.ActorID.Bytes).String()
			actorID = &idStr
		}
		var actorEmail *string
		if l.ActorEmail.Valid {
			actorEmail = &l.ActorEmail.String
		}
		var actorRole *string
		if l.ActorRole.Valid {
			actorRole = &l.ActorRole.String
		}
		var resourceID *string
		if l.ResourceID.Valid {
			resourceID = &l.ResourceID.String
		}

		var metadata map[string]any
		if len(l.Metadata) > 0 {
			_ = json.Unmarshal(l.Metadata, &metadata)
		}

		data[i] = AuditLogResponse{
			ID:           l.ID.String(),
			ActorID:      actorID,
			ActorEmail:   actorEmail,
			ActorRole:    actorRole,
			Action:       l.Action,
			ResourceType: l.ResourceType,
			ResourceID:   resourceID,
			Status:       l.Status,
			IPAddress:    l.IpAddress,
			UserAgent:    l.UserAgent,
			Metadata:     metadata,
			CreatedAt:    l.CreatedAt,
		}
	}

	return PaginatedAuditLogsResponse{
		Data:       data,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
	}, nil
}
