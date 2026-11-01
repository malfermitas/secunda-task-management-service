package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	analyticsapp "secunda-task-management-service/internal/analytics/app"
	"secunda-task-management-service/internal/infrastructure/auth"
	rediscache "secunda-task-management-service/internal/infrastructure/cache/redis"
	"secunda-task-management-service/internal/infrastructure/config"
	"secunda-task-management-service/internal/infrastructure/logging"
	mysqlrepo "secunda-task-management-service/internal/infrastructure/persistence/mysql"
	"secunda-task-management-service/internal/interfaces/http/handler"
	taskapp "secunda-task-management-service/internal/task/app"
	teamapp "secunda-task-management-service/internal/team/app"
	userapp "secunda-task-management-service/internal/user/app"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := logging.New(cfg.Log)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := mysqlrepo.Open(ctx, cfg.DB)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	userRepo := mysqlrepo.NewUserRepository(db)
	teamRepo := mysqlrepo.NewTeamRepository(db)
	taskRepo := mysqlrepo.NewTaskRepository(db)
	statsRepo := mysqlrepo.NewStatsRepository(db)

	hasher := auth.NewBcryptHasher(0)
	tokens, err := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.TTL)
	if err != nil {
		return fmt.Errorf("create jwt manager: %w", err)
	}

	taskCache := rediscache.NewCache(cfg.Redis, cfg.Cache.TTL, logger)
	defer func() {
		if err := taskCache.Close(); err != nil {
			logger.Warn("close redis client", "error", err)
		}
	}()

	router := handler.NewRouter(handler.Dependencies{
		Auth:        userapp.NewAuthService(userRepo, hasher, tokens),
		Teams:       teamapp.NewService(teamRepo, userRepo),
		Tasks:       taskapp.NewService(taskRepo, teamRepo, taskCache),
		Stats:       analyticsapp.NewService(teamRepo, statsRepo),
		TokenParser: tokens,
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.App.Port),
		Handler:           router,
		ReadHeaderTimeout: cfg.App.ReadHeaderTimeout,
		ReadTimeout:       cfg.App.ReadTimeout,
		WriteTimeout:      cfg.App.WriteTimeout,
		IdleTimeout:       cfg.App.IdleTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info(
			"starting api",
			"port", cfg.App.Port,
			"log_level", cfg.Log.Level,
			"log_format", cfg.Log.Format,
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
		return nil
	}
}
