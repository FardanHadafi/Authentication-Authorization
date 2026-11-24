package auth

import (
	"authentication-authorization/shared"
	"net/http"
	"time"

	"github.com/go-redis/redis_rate/v10"
)

func RegisterRoutes(mux *http.ServeMux, handler AuthHandler, tokenManager shared.TokenManager, limiter *redis_rate.Limiter, revocationChecker shared.TokenRevocationChecker) {
	mux.Handle("POST /api/auth/register", shared.Chain(
		http.HandlerFunc(handler.Register),
		shared.RateLimitMiddleware(limiter, shared.IPKey("register"), 10, time.Hour),
	))
	mux.Handle("GET /api/auth/verify", shared.Chain(
		http.HandlerFunc(handler.VerifyEmail),
		shared.RateLimitMiddleware(limiter, shared.IPKey("verify"), 20, time.Hour),
	))
	mux.Handle("POST /api/auth/login", shared.Chain(
		http.HandlerFunc(handler.Login),
		shared.RateLimitMiddleware(limiter, shared.IPKey("login"), 5, 15*time.Minute),
	))
	mux.Handle("POST /api/auth/refresh", shared.Chain(
		http.HandlerFunc(handler.RefreshToken),
		shared.RateLimitMiddleware(limiter, shared.IPKey("refresh"), 30, time.Minute),
	))
	mux.Handle("POST /api/auth/logout", shared.Chain(
		http.HandlerFunc(handler.Logout),
		shared.RateLimitMiddleware(limiter, shared.AutoKey("logout"), 30, time.Minute),
	))
	mux.Handle("POST /api/auth/forgot-password", shared.Chain(
		http.HandlerFunc(handler.ForgotPassword),
		shared.RateLimitMiddleware(limiter, shared.IPKey("forgot-pass"), 3, 15*time.Minute),
	))
	mux.Handle("POST /api/auth/reset-password", shared.Chain(
		http.HandlerFunc(handler.ResetPassword),
		shared.RateLimitMiddleware(limiter, shared.IPKey("reset-pass"), 5, 15*time.Minute),
	))
	mux.Handle("POST /api/auth/resend-verification", shared.Chain(
		http.HandlerFunc(handler.ResendVerificationEmail),
		shared.RateLimitMiddleware(limiter, shared.IPKey("resend-verify"), 3, 15*time.Minute),
	))
	mux.Handle("GET /api/auth/csrf", shared.Chain(
		http.HandlerFunc(handler.GetCSRFToken),
		shared.RateLimitMiddleware(limiter, shared.IPKey("csrf"), 60, time.Minute),
	))

	// Protected Routes
	mux.Handle("GET /api/auth/me", shared.Chain(
		http.HandlerFunc(handler.GetProfile),
		shared.RequireAuth(tokenManager, revocationChecker),
		shared.RateLimitMiddleware(limiter, shared.UserKey("me"), 60, time.Minute),
	))

	// Admin Routes
	mux.Handle("GET /api/admin/users", shared.Chain(
		http.HandlerFunc(handler.ListUsers),
		shared.RequireAuth(tokenManager, revocationChecker),
		shared.RequireRole("admin"),
		shared.RateLimitMiddleware(limiter, shared.UserKey("admin-users"), 30, time.Minute),
	))
	mux.Handle("GET /api/admin/audit-logs", shared.Chain(
		http.HandlerFunc(handler.ListAuditLogs),
		shared.RequireAuth(tokenManager, revocationChecker),
		shared.RequireRole("admin"),
		shared.RateLimitMiddleware(limiter, shared.UserKey("admin-audit-logs"), 60, time.Minute),
	))
}
