package auth

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"authentication-authorization/shared"

	"github.com/go-playground/validator/v10"
)

type AuthHandler interface {
	Register(w http.ResponseWriter, r *http.Request)
	VerifyEmail(w http.ResponseWriter, r *http.Request)
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
