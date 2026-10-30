package domain

import "context"

// Repository is the persistence port for the user aggregate.
type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email Email) (*User, error)
	GetByID(ctx context.Context, userID int64) (*User, error)
}
