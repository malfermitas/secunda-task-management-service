package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager issues and validates HS256 access tokens.
type JWTManager struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTManager creates a JWT manager. The secret must not be empty.
func NewJWTManager(secret string, ttl time.Duration) (*JWTManager, error) {
	if secret == "" {
		return nil, errors.New("jwt secret must not be empty")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &JWTManager{secret: []byte(secret), ttl: ttl}, nil
}

// Issue creates a signed token with the user ID as its subject.
func (m *JWTManager) Issue(userID int64) (string, error) {
	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ParseToken validates a token and returns its user ID.
func (m *JWTManager) ParseToken(tokenString string) (int64, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return 0, err
	}
	if !token.Valid {
		return 0, jwt.ErrTokenInvalidClaims
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return 0, errors.New("invalid token subject")
	}
	return userID, nil
}
