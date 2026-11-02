package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	sharedlogging "secunda-task-management-service/internal/shared/logging"
)

const errorKey = "request_error"

// SetError stores the request error for the completion log.
func SetError(c *gin.Context, err error) {
	c.Set(errorKey, err)
}

// Logging adds request-scoped fields and emits one completion log per request.
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		logger := sharedlogging.WithTrace(
			c.Request.Context(),
			sharedlogging.FromContext(c.Request.Context()),
		).With(
			zap.String("request_id", RequestIDFromContext(c)),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
		)
		c.Request = c.Request.WithContext(
			sharedlogging.WithLogger(c.Request.Context(), logger),
		)

		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		fields := []zap.Field{
			zap.String("route", route),
			zap.Int("status", c.Writer.Status()),
			zap.Int64("latency_ms", time.Since(start).Milliseconds()),
			zap.Int("bytes", max(c.Writer.Size(), 0)),
		}
		if userID, ok := UserID(c); ok {
			fields = append(fields, zap.Int64("user_id", userID))
		}
		if err, ok := c.Get(errorKey); ok {
			if requestErr, ok := err.(error); ok {
				fields = append(fields, zap.Error(requestErr))
			}
		}

		logger = sharedlogging.FromContext(c.Request.Context())
		logger = sharedlogging.WithTrace(c.Request.Context(), logger)

		if c.Request.URL.Path == "/healthz" ||
			strings.HasPrefix(c.Request.URL.Path, "/swagger") {
			logger.Debug("request completed", fields...)
			return
		}

		switch {
		case c.Writer.Status() >= 500:
			logger.Error("request completed", fields...)
		case c.Writer.Status() >= 400:
			logger.Warn("request completed", fields...)
		default:
			logger.Info("request completed", fields...)
		}
	}
}
