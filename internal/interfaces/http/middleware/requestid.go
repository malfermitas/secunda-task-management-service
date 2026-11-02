package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// RequestIDKey is the gin context key that stores the request ID.
const RequestIDKey = "request_id"

const requestIDHeader = "X-Request-ID"

// RequestID accepts an incoming request ID or generates a new one.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(requestIDHeader)
		if !validRequestID(requestID) {
			requestID = newRequestID()
		}

		c.Set(RequestIDKey, requestID)
		c.Header(requestIDHeader, requestID)
		trace.SpanFromContext(c.Request.Context()).SetAttributes(
			attribute.String("http.request_id", requestID),
		)
		c.Next()
	}
}

// RequestIDFromContext returns the request ID stored in the gin context.
func RequestIDFromContext(c *gin.Context) string {
	requestID, _ := c.Get(RequestIDKey)
	value, _ := requestID.(string)
	return value
}

func newRequestID() string {
	var buffer [16]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buffer[:])
}

func validRequestID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < 0x21 || value[i] > 0x7e {
			return false
		}
	}
	return true
}
