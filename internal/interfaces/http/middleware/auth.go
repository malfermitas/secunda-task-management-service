package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	sharedlogging "secunda-task-management-service/internal/shared/logging"
)

// UserIDKey is the gin context key that stores the authenticated user ID.
const UserIDKey = "user_id"

// TokenParser validates a bearer token and returns the user ID it belongs to.
type TokenParser interface {
	ParseToken(token string) (int64, error)
}

// Auth returns middleware that requires a valid bearer token.
func Auth(parser TokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		if parser == nil {
			abortUnauthorized(c, "authentication is not configured")
			return
		}

		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			abortUnauthorized(c, "missing or invalid authorization header")
			return
		}

		userID, err := parser.ParseToken(token)
		if err != nil || userID <= 0 {
			abortUnauthorized(c, "invalid or expired token")
			return
		}

		c.Set(UserIDKey, userID)
		ctx := sharedlogging.WithLogger(
			c.Request.Context(),
			sharedlogging.FromContext(c.Request.Context()).With(zap.Int64("user_id", userID)),
		)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// UserID returns the authenticated user ID from the gin context.
func UserID(c *gin.Context) (int64, bool) {
	value, exists := c.Get(UserIDKey)
	if !exists {
		return 0, false
	}

	userID, ok := value.(int64)
	if !ok || userID <= 0 {
		return 0, false
	}
	return userID, true
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}

	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}

func abortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{
			"code":    "unauthorized",
			"message": message,
		},
	})
}
