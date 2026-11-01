package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	analyticsapp "secunda-task-management-service/internal/analytics/app"
	"secunda-task-management-service/internal/infrastructure/auth"
	"secunda-task-management-service/internal/infrastructure/config"
	mysqlrepo "secunda-task-management-service/internal/infrastructure/persistence/mysql"
	"secunda-task-management-service/internal/interfaces/http/handler"
	taskapp "secunda-task-management-service/internal/task/app"
	teamapp "secunda-task-management-service/internal/team/app"
	userapp "secunda-task-management-service/internal/user/app"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := mysqlrepo.Open(ctx, cfg.DB)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	userRepo := mysqlrepo.NewUserRepository(db)
	teamRepo := mysqlrepo.NewTeamRepository(db)
	taskRepo := mysqlrepo.NewTaskRepository(db)
	statsRepo := mysqlrepo.NewStatsRepository(db)

	hasher := auth.NewBcryptHasher(0)
	tokens, err := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.TTL)
	if err != nil {
		slog.Error("create jwt manager", "error", err)
		os.Exit(1)
	}

	router := handler.NewRouter(handler.Dependencies{
		Auth:        userapp.NewAuthService(userRepo, hasher, tokens),
		Teams:       teamapp.NewService(teamRepo, userRepo),
		Tasks:       taskapp.NewService(taskRepo, teamRepo, nil),
		Stats:       analyticsapp.NewService(teamRepo, statsRepo),
		TokenParser: tokens,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("starting api", "port", cfg.AppPort)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
