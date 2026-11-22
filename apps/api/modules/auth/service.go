package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
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
)

type AuthService interface {
	Register(ctx context.Context, request CreateUserRequest) (UserResponse, error)
	VerifyEmail(ctx context.Context, token string) error
	Login(ctx context.Context, request LoginUserRequest) (AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (AuthResponse, error)
	Logout(ctx context.Context, refreshToken string) error
}

type authService struct {
	querier      db.Querier
	asynqClient  *asynq.Client
	mailer       shared.EmailSender
	redis        *redis.Client
	tokenManager shared.TokenManager
}

func NewAuthService(querier db.Querier, asynqClient *asynq.Client, mailer shared.EmailSender, redis *redis.Client, tokenManager shared.TokenManager) AuthService {
	return &authService{
		querier:      querier,
		asynqClient:  asynqClient,
		mailer:       mailer,
		redis:        redis,
		tokenManager: tokenManager,
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
	if taskErr == nil && a.asynqClient != nil {
		_, _ = a.asynqClient.EnqueueContext(ctx, task)
	}

	// 6. Return response yang sudah dipetakan dengan DTO
	return UserResponse{
		ID:        newUser.ID.String(),
		FullName:  newUser.FullName,
		Email:     newUser.Email,
		Role:      newUser.Role,
		Status:    newUser.Status,
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
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
		return AuthResponse{}, ErrInvalidCredentials
	}

	// Pastikan status akun sudah aktif
	if user.Status == "pending" {
		return AuthResponse{}, ErrAccountNotVerified
	} else if user.Status != "active" {
		return AuthResponse{}, errors.New("account is not active yet, please check your verification link in your mailbox")
	}

	// Terbitkan AccessToken
	accessToken, expiresAt, err := a.tokenManager.GenerateAccessToken(user.ID, user.Email, user.Role)
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

	// Return AuthResponse
	return AuthResponse{
		TokenType:    "Bearer",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User: UserResponse{
			ID:        user.ID.String(),
			FullName:  user.FullName,
			Email:     user.Email,
			Role:      user.Role,
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
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
	accessToken, expiresAt, err := a.tokenManager.GenerateAccessToken(user.ID, user.Email, user.Role)
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
			ID:        user.ID.String(),
			FullName:  user.FullName,
			Email:     user.Email,
			Role:      user.Role,
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
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

	return nil
}
