package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"secunda-task-management-service/internal/interfaces/http/handler"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	router := handler.NewRouter(handler.Dependencies{})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("starting api", "port", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
