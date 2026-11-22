package auth

import "net/http"

func RegisterRoutes(mux *http.ServeMux, handler AuthHandler) {
	mux.HandleFunc("POST /api/auth/register", handler.Register)
	mux.HandleFunc("GET /api/auth/verify", handler.VerifyEmail)
	mux.HandleFunc("POST /api/auth/login", handler.Login)
	mux.HandleFunc("POST /api/auth/refresh", handler.RefreshToken)
	mux.HandleFunc("POST /api/auth/logout", handler.Logout)
}
