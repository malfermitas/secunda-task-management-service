package domain

import (
	"fmt"
	"net/mail"
	"strings"
	"time"

	"secunda-task-management-service/internal/shared/errs"
)

// Email is a value object representing a normalized email address.
type Email string

// NewEmail normalizes and validates an email address.
func NewEmail(raw string) (Email, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "", errs.Validation("email is required")
	}

	addr, err := mail.ParseAddress(normalized)
	if err != nil || addr.Address != normalized || !strings.Contains(normalized, "@") {
		return "", errs.Validation("invalid email")
	}
	return Email(normalized), nil
}

// String returns the email as a plain string.
func (e Email) String() string {
	return string(e)
}

// User is the user aggregate root.
type User struct {
	ID           int64
	Email        Email
	PasswordHash string
	Name         string
	CreatedAt    time.Time
}

// NewUser creates a validated user entity.
func NewUser(email Email, passwordHash, name string, now time.Time) (*User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errs.Validation("name is required")
	}
	if len(name) > 255 {
		return nil, errs.Validation("name must be at most 255 characters")
	}
	if passwordHash == "" {
		return nil, errs.Validation("password hash is required")
	}

	return &User{
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
		CreatedAt:    now.UTC(),
	}, nil
}

// ErrNotFound is returned when a user does not exist.
func ErrNotFound(id int64) *errs.Error {
	return errs.NotFound("user_not_found", fmt.Sprintf("user %d not found", id))
}

// ErrEmailTaken is returned when the email is already registered.
func ErrEmailTaken() *errs.Error {
	return errs.Conflict("email_taken", "email is already registered")
}
