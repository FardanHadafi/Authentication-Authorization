package auth

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"authentication-authorization/shared"

	"github.com/go-playground/validator/v10"
)

type AuthHandler interface {
	Register(w http.ResponseWriter, r *http.Request)
	VerifyEmail(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
	RefreshToken(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
	ForgotPassword(w http.ResponseWriter, r *http.Request)
	ResetPassword(w http.ResponseWriter, r *http.Request)
	ResendVerificationEmail(w http.ResponseWriter, r *http.Request)
	GetProfile(w http.ResponseWriter, r *http.Request)
	ListUsers(w http.ResponseWriter, r *http.Request)
	ListAuditLogs(w http.ResponseWriter, r *http.Request)
	GetCSRFToken(w http.ResponseWriter, r *http.Request)
}

type authHandler struct {
	authService AuthService
	logger      *slog.Logger
	validator   *validator.Validate
}

func NewAuthHandler(authService AuthService, logger *slog.Logger, validator *validator.Validate) AuthHandler {
	return &authHandler{
		authService: authService,
		logger:      logger,
		validator:   validator,
	}
}

func (h *authHandler) Register(w http.ResponseWriter, r *http.Request) {
	var request CreateUserRequest

	// 1. Decode request body menggunakan encoding/json/v2
	if err := json.UnmarshalRead(r.Body, &request); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 2. Validasi user request struct
	if err := h.validator.StructCtx(r.Context(), request); err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 3. Panggil service untuk mendaftarkan user
	user, err := h.authService.Register(r.Context(), request)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			shared.WriteError(w, http.StatusConflict, err.Error())
			return
		}

		h.logger.Error("failed to register user", "error", err)
		shared.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// 4. Kirim response sukses (201 Created)
	_ = shared.WriteJSON(w, http.StatusCreated, user)
}

func (h *authHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		shared.WriteError(w, http.StatusBadRequest, "token is required")
		return
	}

	if err := h.authService.VerifyEmail(r.Context(), token); err != nil {
		if errors.Is(err, ErrInvalidOrExpiredToken) {
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		h.logger.Error("failed to verify email", "error", err)
		shared.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Redirect ke frontend SvelteKit dengan query verified=true
	frontendURL := strings.TrimRight(os.Getenv("FRONTEND_URL"), "/")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}
	http.Redirect(w, r, fmt.Sprintf("%s/register?verified=true", frontendURL), http.StatusSeeOther)
}

func (h *authHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request LoginUserRequest

	// Decode user request
	if err := json.UnmarshalRead(r.Body, &request); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validasi user request struct
	if err := h.validator.StructCtx(r.Context(), request); err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Panggil service
	authResponse, err := h.authService.Login(r.Context(), request)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			shared.WriteError(w, http.StatusUnauthorized, err.Error())
			return
		}

		if errors.Is(err, ErrAccountNotVerified) {
			shared.WriteError(w, http.StatusForbidden, err.Error())
			return
		}

		h.logger.Error("failed to login", "error", err)
		shared.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Pasang access dan refresh token ke dalam httpOnly cookie
	isProd := os.Getenv("APP_ENV") == "production"
	shared.SetAuthCookie(w, authResponse.AccessToken, authResponse.RefreshToken, isProd)

	// Kirim response 200 Ok dengan payload AuthResponse
	_ = shared.WriteJSON(w, http.StatusOK, authResponse)
}

func (h *authHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var token string

	// 1. Coba baca dari request body jika ada
	if r.Body != nil && r.ContentLength > 0 {
		var request RefreshTokenRequest
		if err := json.UnmarshalRead(r.Body, &request); err == nil {
			token = request.RefreshToken
		}
	}

	// 2. Fallback baca dari httpOnly cookie jika tidak disertakan di body
	if strings.TrimSpace(token) == "" {
		if cookie, err := r.Cookie("refresh_token"); err == nil {
			token = cookie.Value
		}
	}

	token = strings.TrimSpace(token)
	if token == "" {
		shared.WriteError(w, http.StatusBadRequest, "refresh token is required")
		return
	}

	authResponse, err := h.authService.RefreshToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, ErrInvalidOrExpiredToken) {
			shared.WriteError(w, http.StatusUnauthorized, "invalid or expired refresh token")
			return
		}

		h.logger.Error("failed to refresh token", "error", err)
		shared.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Pasang access dan refresh token baru ke httpOnly cookie (RTR)
	isProd := os.Getenv("APP_ENV") == "production"
	shared.SetAuthCookie(w, authResponse.AccessToken, authResponse.RefreshToken, isProd)

	_ = shared.WriteJSON(w, http.StatusOK, authResponse)
}

func (h *authHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var token string

	// 1. Coba baca dari request body jika ada
	if r.Body != nil && r.ContentLength > 0 {
		var request LogoutRequest
		if err := json.UnmarshalRead(r.Body, &request); err == nil {
			token = request.RefreshToken
		}
	}

	// 2. Fallback baca dari httpOnly cookie jika tidak disertakan di body
	if strings.TrimSpace(token) == "" {
		if cookie, err := r.Cookie("refresh_token"); err == nil {
			token = cookie.Value
		}
	}

	// 3. Invalidate refresh token di Redis
	if strings.TrimSpace(token) != "" {
		if err := h.authService.Logout(r.Context(), token); err != nil {
			h.logger.Error("failed to revoke refresh token during logout", "error", err)
		}
	}

	// 4. Hapus cookie dari browser client
	isProd := os.Getenv("APP_ENV") == "production"
	shared.ClearAuthCookie(w, isProd)

	_ = shared.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

func (h *authHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var request ForgotPasswordRequest

	if err := json.UnmarshalRead(r.Body, &request); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validator.StructCtx(r.Context(), request); err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.authService.ForgotPassword(r.Context(), request); err != nil {
		h.logger.Error("failed to process forgot password", "error", err)
		shared.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	_ = shared.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Jika email terdaftar di sistem kami, instruksi reset kata sandi telah dikirimkan ke kotak masuk Anda.",
	})
}

func (h *authHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var request ResetPasswordRequest

	// 1. Decode request body
	if err := json.UnmarshalRead(r.Body, &request); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 2. Validasi input struct (token, new_password, confirm_new_password)
	if err := h.validator.StructCtx(r.Context(), request); err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 3. Panggil service ResetPassword
	if err := h.authService.ResetPassword(r.Context(), request); err != nil {
		if errors.Is(err, ErrInvalidOrExpiredToken) {
			shared.WriteError(w, http.StatusBadRequest, "invalid or expired reset token")
			return
		}

		h.logger.Error("failed to reset password", "error", err)
		shared.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// 4. Bersihkan cookie auth yang mungkin ada di peramban
	isProd := os.Getenv("APP_ENV") == "production"
	shared.ClearAuthCookie(w, isProd)

	// 5. Kembalikan response 200 OK
	_ = shared.WriteJSON(w, http.StatusOK, ResetPasswordResponse{
		Message: "password has been reset successfully, please login with your new password",
	})
}

func (h *authHandler) ResendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	var request ResendVerificationEmailRequest

	if err := json.UnmarshalRead(r.Body, &request); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validator.StructCtx(r.Context(), request); err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.authService.ResendVerificationEmail(r.Context(), request); err != nil {
		h.logger.Error("failed to resend verification email", "error", err)
		shared.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	_ = shared.WriteJSON(w, http.StatusOK, ResendVerificationEmailResponse{
		Message: "Jika email terdaftar dan belum diverifikasi, tautan verifikasi baru telah dikirimkan ke kotak masuk Anda.",
	})
}

func (h *authHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Ambil claims yang sudah di inject oleh RequireAuth middleware
	claims := shared.GetUserClaims(r.Context())
	if claims == nil {
		shared.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Panggil service dengan claims.UserID
	user, err := h.authService.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			shared.WriteError(w, http.StatusNotFound, "user not found")
			return
		}

		if errors.Is(err, ErrAccountInactive) {
			shared.WriteError(w, http.StatusForbidden, "account is inactive or suspended")
			return
		}

		h.logger.Error("failed to get user profile", "error", err)
		shared.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	_ = shared.WriteJSON(w, http.StatusOK, user)
}

func (h *authHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.authService.ListUsers(r.Context())
	if err != nil {
		h.logger.Error("failed to list users", "error", err)
		shared.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	_ = shared.WriteJSON(w, http.StatusOK, users)
}

func (h *authHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	page := int32(1)
	limit := int32(20)

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = int32(p)
		}
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = int32(l)
		}
	}

	res, err := h.authService.ListAuditLogs(r.Context(), page, limit)
	if err != nil {
		h.logger.Error("failed to list audit logs", "error", err)
		shared.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	_ = shared.WriteJSON(w, http.StatusOK, res)
}

func (h *authHandler) GetCSRFToken(w http.ResponseWriter, r *http.Request) {
	token, err := shared.GenerateCSRFToken()
	if err != nil {
		h.logger.Error("failed to generate csrf token", "error", err)
		shared.WriteError(w, http.StatusInternalServerError, "failed to generate csrf token")
		return
	}

	isProd := os.Getenv("APP_ENV") == "production"
	shared.SetCSRFCookie(w, token, isProd)
	_ = shared.WriteJSON(w, http.StatusOK, CSRFTokenResponse{CSRFToken: token})
}
