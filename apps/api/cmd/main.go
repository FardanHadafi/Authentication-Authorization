package main

import (
	"authentication-authorization/modules/auth"
	"authentication-authorization/shared"
	"authentication-authorization/shared/db"
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/go-redis/redis_rate/v10"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func main() {
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := shared.InitLogger()

	cfg, err := shared.LoadConfig()
	if err != nil {
		logger.Error("Failed to load environment configuration", "error", err)
		os.Exit(1)
	}

	database, err := shared.InitDatabase(rootCtx, cfg.DBURL)
	if err != nil {
		logger.Error("Cannot init database", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	logger.Info("Database connected successfully !")

	queries := db.New(database)
	transactor := shared.NewTransactor(database)

	// Mailer with go-mail & circuit breaker
	mailer := shared.NewMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom)

	// Asynq Client
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr})
	defer asynqClient.Close()

	// Redis Client (untuk token verifikasi & cache)
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	defer redisClient.Close()

	// Asynq Server
	asynqServer := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisAddr},
		asynq.Config{Concurrency: 10},
	)

	asynqMux := asynq.NewServeMux()
	taskHandler := shared.NewTaskHandler(mailer, logger)
	taskHandler.Register(asynqMux)

	// Asynq Worker Goroutine
	go func() {
		if err := asynqServer.Run(asynqMux); err != nil {
			logger.Error("Asynq worker failed", "error", err)
		}
	}()
	defer asynqServer.Shutdown()

	validate := validator.New(validator.WithRequiredStructEnabled())

	tokenManager := shared.NewTokenManager([]byte(cfg.JWTSecret), 15*time.Minute, "crackerstudio@digital")

	mux := http.NewServeMux()

	// Audit Logger (Asynchronous non-blocking security audit logger)
	auditLogger := shared.NewAuditLogger(queries, logger, 1000)

	// Permission Checker (PBAC with Redis caching)
	permissionChecker := shared.NewPermissionChecker(queries, redisClient)

	// Auth Service (equipped with database Transactor for atomic mutations)
	authService := auth.NewAuthService(queries, transactor, asynqClient, mailer, redisClient, tokenManager, auditLogger, permissionChecker)

	// Auth Handler
	authHandler := auth.NewAuthHandler(authService, logger, validate)

	// Redis Rate Limiter
	limiter := redis_rate.NewLimiter(redisClient)

	// Token Revocation Checker
	revocationChecker := shared.NewRevocationChecker(redisClient)

	// Auth Routes
	auth.RegisterRoutes(mux, authHandler, tokenManager, limiter, revocationChecker, permissionChecker)

	// Prometheus Metrics Endpoint
	mux.Handle("GET /metrics", promhttp.Handler())

	// OpenTelemetry Tracing ke Grafana Tempo
	_, shutdownTracer, err := shared.InitTracer(rootCtx, "auth-api", cfg.TempoEndpoint)
	if err != nil {
		logger.Warn("Failed to initialize OpenTelemetry tracer, continuing without tracing", "error", err)
	} else {
		defer func() {
			if err := shutdownTracer(context.Background()); err != nil {
				logger.Error("Failed to shutdown tracer gracefully", "error", err)
			}
		}()
		logger.Info("OpenTelemetry tracer initialized successfully", "endpoint", cfg.TempoEndpoint)
	}

	port := cfg.APIPort
	if port == "" {
		port = "8080"
	}

	csrfConfig := shared.DefaultCSRFConfig()

	router := shared.Chain(
		mux,
		shared.RequestIDMiddleware,
		shared.TracingMiddleware("auth-api"),
		shared.MetricsMiddleware,
		shared.RecoveryMiddleware(logger),
		shared.LoggingMiddleware(logger),
		shared.CSRFMiddleware(csrfConfig),
	)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  1 * time.Minute,
	}

	// HTTP Server Goroutine supaya tidak memblokir sinyal shutdown
	go func() {
		logger.Info("Server is running", "port", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server failed to start", "error", err)
		}
		os.Exit(1)
	}()

	// Tunggu sinyal shutdown
	<-rootCtx.Done()
	logger.Info("Shutdown signal received, starting graceful shutdown...")

	// Beri waktu 10 detik untuk menyelesaikan semua request & task yang sedang berjalan
	shutdownCtx, cancel := context.WithTimeout(rootCtx, 10*time.Second)
	defer cancel()

	// 1. Matikan HTTP Server, tutup koneksi baru, selesaikan request aktif
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server forced to shutdown", "error", err)
	} else {
		logger.Info("HTTP server stopped gracefully")
	}

	// 2. Matikan Asynq Server, tunggu task email yang sedang dikirim selesai
	asynqServer.Shutdown()
	logger.Info("Asynq worker stopped gracefully")

	// 3. Matikan Audit Logger, flush sisa antrean log audit ke PostgreSQL
	if err := auditLogger.Shutdown(shutdownCtx); err != nil {
		logger.Error("Audit logger forced to shutdown", "error", err)
	} else {
		logger.Info("Audit logger flushed and stopped gracefully")
	}

	logger.Info("Application exited cleanly")
}
