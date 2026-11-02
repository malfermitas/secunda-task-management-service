package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDGeneratesAndReturnsHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RequestID())
	engine.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	requestID := response.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Fatal("X-Request-ID header is empty")
	}
	if len(requestID) != 32 {
		t.Errorf("X-Request-ID length = %d, want 32", len(requestID))
	}
}

func TestRequestIDPreservesValidIncomingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RequestID())
	engine.GET("/", func(c *gin.Context) {
		if got := RequestIDFromContext(c); got != "client-request-1" {
			t.Errorf("RequestIDFromContext() = %q, want client-request-1", got)
		}
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "client-request-1")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if got := response.Header().Get("X-Request-ID"); got != "client-request-1" {
		t.Errorf("X-Request-ID = %q, want client-request-1", got)
	}
}

func TestRequestIDRejectsInvalidIncomingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RequestID())
	engine.GET("/", func(c *gin.Context) {
		if got := RequestIDFromContext(c); got == "bad id" {
			t.Error("invalid request ID was accepted")
		}
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "bad id")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Header().Get("X-Request-ID") == "bad id" {
		t.Fatal("invalid request ID was returned")
	}
}
