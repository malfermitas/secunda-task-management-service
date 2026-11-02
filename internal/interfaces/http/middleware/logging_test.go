package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggingRecordsRequestFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	previous := zap.L()
	zap.ReplaceGlobals(logger)
	t.Cleanup(func() { zap.ReplaceGlobals(previous) })

	engine := gin.New()
	engine.Use(RequestID())
	engine.Use(Logging())
	engine.GET("/tasks/:id", func(c *gin.Context) {
		SetError(c, errors.New("boom"))
		c.Status(http.StatusInternalServerError)
	})

	request := httptest.NewRequest(http.MethodGet, "/tasks/42", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	entries := logs.FilterMessage("request completed").All()
	if len(entries) != 1 {
		t.Fatalf("completion logs = %d, want 1", len(entries))
	}

	fields := entries[0].ContextMap()
	if fields["route"] != "/tasks/:id" {
		t.Errorf("route = %v, want /tasks/:id", fields["route"])
	}
	if fields["status"] != int64(http.StatusInternalServerError) {
		t.Errorf("status = %v, want 500", fields["status"])
	}
	if fields["request_id"] == "" {
		t.Error("request_id is empty")
	}
	if fields["error"] == nil {
		t.Error("error field is missing")
	}
	entry := entries[0]
	if entry.Level != zap.ErrorLevel {
		t.Errorf("level = %s, want error", entry.Level)
	}
}
