package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	sharedlogging "secunda-task-management-service/internal/shared/logging"
	sharedtracing "secunda-task-management-service/internal/shared/tracing"
)

// Recovery converts panics into logged 500 responses without leaking details.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			ctx := c.Request.Context()
			err := fmt.Errorf("panic: %v", recovered)
			sharedtracing.RecordError(trace.SpanFromContext(ctx), err)
			sharedlogging.FromContext(ctx).Error(
				"panic recovered",
				zap.Any("panic", recovered),
				zap.ByteString("stack", debug.Stack()),
			)

			if c.Writer.Written() {
				c.Abort()
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "internal_error",
					"message": "internal server error",
				},
			})
		}()

		c.Next()
	}
}
