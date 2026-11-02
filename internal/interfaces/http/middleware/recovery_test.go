package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestRecoveryReturnsInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	previous := zap.L()
	zap.ReplaceGlobals(logger)
	t.Cleanup(func() { zap.ReplaceGlobals(previous) })

	engine := gin.New()
	engine.Use(RequestID())
	engine.Use(Logging())
	engine.Use(Recovery())
	engine.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if response.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", response.Code)
	}
	if logs.FilterMessage("panic recovered").Len() != 1 {
		t.Fatal("panic was not logged")
	}
}
