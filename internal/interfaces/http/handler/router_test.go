package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSwaggerDocumentIsServed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{ServiceName: "test-service"})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if !strings.Contains(response.Body.String(), "Secunda Task Management API") {
		t.Fatal("swagger document does not contain API title")
	}
}
