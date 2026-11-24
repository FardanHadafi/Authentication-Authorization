package auth

import (
	"authentication-authorization/shared"
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux, handler AuthHandler, tokenManager shared.TokenManager) {
	mux.HandleFunc("POST /api/auth/register", handler.Register)
	mux.HandleFunc("GET /api/auth/verify", handler.VerifyEmail)
	mux.HandleFunc("POST /api/auth/login", handler.Login)
	mux.HandleFunc("POST /api/auth/refresh", handler.RefreshToken)
	mux.HandleFunc("POST /api/auth/logout", handler.Logout)
	mux.HandleFunc("POST /api/auth/forgot-password", handler.ForgotPassword)
	mux.HandleFunc("POST /api/auth/reset-password", handler.ResetPassword)
	mux.HandleFunc("POST /api/auth/resend-verification", handler.ResendVerificationEmail)

	// Protected Routes
	mux.Handle("GET /api/auth/me", shared.Chain(http.HandlerFunc(handler.GetProfile), shared.RequireAuth(tokenManager)))

	// Admin Routes
	mux.Handle("GET /api/admin/users", shared.Chain(http.HandlerFunc(handler.ListUsers), shared.RequireAuth(tokenManager), shared.RequireRole("admin")))
}
