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
	ErrEmailAlreadyExists     = errors.New("email already registered")
	ErrInvalidOrExpiredToken = errors.New("invalid or expired verification token")
)

type AuthService interface {
	Register(ctx context.Context, request CreateUserRequest) (UserResponse, error)
	VerifyEmail(ctx context.Context, token string) error
}

type authService struct {
	querier     db.Querier
	asynqClient *asynq.Client
	mailer      shared.EmailSender
	redis       *redis.Client
}

func NewAuthService(querier db.Querier, asynqClient *asynq.Client, mailer shared.EmailSender, redis *redis.Client) AuthService {
	return &authService{
		querier:     querier,
		asynqClient: asynqClient,
		mailer:      mailer,
		redis:       redis,
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
		return errors.New("redis client not initialized")
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
