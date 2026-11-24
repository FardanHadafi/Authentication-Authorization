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
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func main() {
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := shared.InitLogger()

	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found or cannot load it", "error", err)
	}

	connectionString := os.Getenv("DB_URL")
	if connectionString == "" {
		logger.Error("DB_URL environment variable is required")
		os.Exit(1)
	}

	database, err := shared.InitDatabase(rootCtx, connectionString)
	if err != nil {
		logger.Error("Cannot init database", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	logger.Info("Database connected successfully !")

	queries := db.New(database)

	// Mailpit
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpFrom := os.Getenv("SMTP_FROM")
	if smtpFrom == "" {
		smtpFrom = os.Getenv("FROM")
	}
	if smtpFrom == "" {
		smtpFrom = "crackerstudio@digital"
	}
	mailer := shared.NewMailer(smtpHost, smtpPort, smtpFrom)

	// Asynq Client
	redisAddress := os.Getenv("REDIS_ADDR")
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddress})
	defer asynqClient.Close()

	// Redis Client (untuk token verifikasi & cache)
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddress,
	})
	defer redisClient.Close()

	// Asynq Server
	asynqServer := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddress},
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

	JWT_SECRET := os.Getenv("JWT_SECRET")
	if JWT_SECRET == "" {
		logger.Error("JWT_SECRET is not set in .env file")
		os.Exit(1)
	}

	tokenManager := shared.NewTokenManager([]byte(JWT_SECRET), 15*time.Minute, "crackerstudio@digital")

	mux := http.NewServeMux()

	// Audit Logger (Asynchronous non-blocking security audit logger)
	auditLogger := shared.NewAuditLogger(queries, logger, 1000)

	// Permission Checker (PBAC with Redis caching)
	permissionChecker := shared.NewPermissionChecker(queries, redisClient)

	// Auth Service
	authService := auth.NewAuthService(queries, asynqClient, mailer, redisClient, tokenManager, auditLogger, permissionChecker)

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
	tempoEndpoint := os.Getenv("TEMPO_ENDPOINT")
	if tempoEndpoint == "" {
		tempoEndpoint = "127.0.0.1:4317"
	}

	_, shutdownTracer, err := shared.InitTracer(rootCtx, "auth-api", tempoEndpoint)
	if err != nil {
		logger.Warn("Failed to initialize OpenTelemetry tracer, continuing without tracing", "error", err)
	} else {
		defer func() {
			if err := shutdownTracer(context.Background()); err != nil {
				logger.Error("Failed to shutdown tracer gracefully", "error", err)
			}
		}()
		logger.Info("OpenTelemetry tracer initialized successfully", "endpoint", tempoEndpoint)
	}

	port := os.Getenv("API_PORT")
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
