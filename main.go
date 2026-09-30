package main

import (
	repository "bank-transfer-api/Repository"
	service "bank-transfer-api/Service"
	"bank-transfer-api/database"
	"bank-transfer-api/handler"
	"bank-transfer-api/middleware"
	"bank-transfer-api/observability"
	"context"
	"net/http"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	logger := observability.NewLogger()

	// Initialize telemetry
	ctx := context.Background()
	shutdownTelemetry, err := observability.Setup(ctx)
	if err != nil {
		logger.Error("telemetry initialization failed", "error", err)
		return
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := shutdownTelemetry(shutdownCtx); err != nil {
			logger.Error("telemetry shutdown failed", "error", err)
		}
	}()

	// Initialize metrics
	metrics, err := observability.NewMetrics(ctx)
	if err != nil {
		logger.Error("metrics initialization failed", "error", err)
		return
	}

	db, err := database.Connect()
	if err != nil {
		logger.Error("database connection failed", "error", err)
		return
	}
	defer db.Close()

	// Start monitoring database pool every 30 seconds
	observability.MonitorDatabasePool(db, logger, 30*time.Second)

	// Start alerting on database pool problems every 10 seconds
	observability.AlertOnDatabasePoolProblems(db, logger, 10*time.Second)

	userRepository := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepository)

	accountRepository := repository.NewAccountRepository(db)
	idempotencyRepository := repository.NewIdempotencyRepository(db)

	transferService := service.NewTransferService(
		db,
		accountRepository,
		idempotencyRepository,
		metrics,
	)

	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/health/live", handler.LivenessHandler)
	mux.HandleFunc("/health/ready", handler.ReadinessHandler(db))

	// API endpoints
	mux.HandleFunc("/register", handler.RegisterHandler(authService))
	mux.HandleFunc("/login", handler.LoginHandler(authService))
	mux.Handle(
		"/transfer",
		middleware.JWTMiddleware(
			handler.TransferHandler(transferService),
		),
	)
	mux.Handle(
		"/getAllTransfer",
		middleware.JWTMiddleware(
			handler.GetAllTransfer(transferService),
		),
	)

	handler := middleware.Tracing(
		middleware.Logging(logger, mux),
	)

	server := &http.Server{
		Addr:    ":7070",
		Handler: handler,
	}

	logger.Info("server started", "address", server.Addr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped", "error", err)
	}
}
