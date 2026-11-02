package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	analyticsapp "secunda-task-management-service/internal/analytics/app"
	"secunda-task-management-service/internal/infrastructure/auth"
	rediscache "secunda-task-management-service/internal/infrastructure/cache/redis"
	"secunda-task-management-service/internal/infrastructure/config"
	"secunda-task-management-service/internal/infrastructure/logging"
	"secunda-task-management-service/internal/infrastructure/observability"
	mysqlrepo "secunda-task-management-service/internal/infrastructure/persistence/mysql"
	"secunda-task-management-service/internal/interfaces/http/handler"
	taskapp "secunda-task-management-service/internal/task/app"
	teamapp "secunda-task-management-service/internal/team/app"
	userapp "secunda-task-management-service/internal/user/app"
)

// @title						Secunda Task Management API
// @version					1.0
// @description				REST API for team task management with role-based access, task history, Redis list caching and SQL analytics.
// @description				Authenticate with `POST /api/v1/login`, then send `Authorization: Bearer <token>` for protected endpoints.
// @host						localhost:8080
// @BasePath					/api/v1
// @schemes					http https
// @contact.name				Secunda API Support
// @contact.email				support@example.com
// @license.name				MIT
//
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Type "Bearer" followed by a space and the JWT token.
func main() {
	gin.SetMode(gin.ReleaseMode)

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "application failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger, err := logging.New(cfg.Log, cfg.App.Name, cfg.App.Env, cfg.App.Version)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	zap.ReplaceGlobals(logger)
	defer func() {
		if err := logger.Sync(); err != nil {
			logger.Warn("sync logger failed", zap.Error(err))
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	telemetry, err := observability.New(ctx, cfg.App, cfg.Metrics, cfg.Tracing, logger)
	if err != nil {
		return fmt.Errorf("create observability: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
		defer cancel()
		if err := telemetry.Shutdown(shutdownCtx); err != nil {
			logger.Warn("shutdown observability failed", zap.Error(err))
		}
	}()

	db, err := mysqlrepo.Open(ctx, cfg.DB, logger, telemetry.TracerProvider, telemetry.MeterProvider)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	userRepo := mysqlrepo.NewUserRepository(db.DB)
	teamRepo := mysqlrepo.NewTeamRepository(db.DB)
	taskRepo := mysqlrepo.NewTaskRepository(db.DB)
	statsRepo := mysqlrepo.NewStatsRepository(db.DB)

	hasher := auth.NewBcryptHasher(0)
	tokens, err := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.TTL)
	if err != nil {
		return fmt.Errorf("create jwt manager: %w", err)
	}

	taskCache := rediscache.NewCache(
		cfg.Redis,
		cfg.Cache.TTL,
		rediscache.Dependencies{
			Logger:         logger,
			TracerProvider: telemetry.TracerProvider,
			MeterProvider:  telemetry.MeterProvider,
			Metrics:        telemetry.Metrics,
		},
	)
	defer func() {
		if err := taskCache.Close(); err != nil {
			logger.Warn("close redis client", zap.Error(err))
		}
	}()

	router := handler.NewRouter(handler.Dependencies{
		Auth:        userapp.NewAuthService(userRepo, hasher, tokens),
		Teams:       teamapp.NewService(teamRepo, userRepo),
		Tasks:       taskapp.NewService(taskRepo, teamRepo, taskCache),
		Stats:       analyticsapp.NewService(teamRepo, statsRepo),
		TokenParser: tokens,
		ServiceName: cfg.App.Name,
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.App.Port),
		Handler:           router,
		ReadHeaderTimeout: cfg.App.ReadHeaderTimeout,
		ReadTimeout:       cfg.App.ReadTimeout,
		WriteTimeout:      cfg.App.WriteTimeout,
		IdleTimeout:       cfg.App.IdleTimeout,
	}

	metricsServer := startMetricsServer(logger, cfg.Metrics, telemetry.MetricsHandler())

	serverErr := make(chan error, 1)
	go func() {
		logger.Info(
			"starting api",
			zap.Int("port", cfg.App.Port),
			zap.String("log_level", cfg.Log.Level),
			zap.String("log_format", cfg.Log.Format),
			zap.Bool("metrics_enabled", cfg.Metrics.Enabled),
			zap.String("metrics_addr", cfg.Metrics.Addr),
			zap.Bool("tracing_enabled", cfg.Tracing.Enabled),
			zap.String("tracing_endpoint", cfg.Tracing.Endpoint),
		)
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	case <-ctx.Done():
		logger.Info("shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: %w", err)
		}
		if metricsServer != nil {
			if err := metricsServer.Shutdown(shutdownCtx); err != nil {
				logger.Warn("shutdown metrics server failed", zap.Error(err))
			}
		}
		return nil
	}
}

func startMetricsServer(
	logger *zap.Logger,
	cfg config.Metrics,
	handler http.Handler,
) *http.Server {
	if !cfg.Enabled || handler == nil {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle(cfg.Path, handler)
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("metrics server stopped", zap.Error(err))
		}
	}()
	return server
}
