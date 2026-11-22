package auth

import "net/http"

func RegisterRoutes(mux *http.ServeMux, handler AuthHandler) {
	mux.HandleFunc("POST /api/auth/register", handler.Register)
	mux.HandleFunc("GET /api/auth/verify", handler.VerifyEmail)
	mux.HandleFunc("GET /verify", handler.VerifyEmail)
}
