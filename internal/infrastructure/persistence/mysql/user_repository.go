package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	userdomain "secunda-task-management-service/internal/user/domain"
)

// UserRepository is the MySQL adapter for the user aggregate.
type UserRepository struct {
	db *sqlx.DB
}

var _ userdomain.Repository = (*UserRepository)(nil)

// NewUserRepository creates a user repository.
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

type userRow struct {
	ID           int64     `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	Name         string    `db:"name"`
	CreatedAt    time.Time `db:"created_at"`
}

func (row userRow) toDomain() *userdomain.User {
	return &userdomain.User{
		ID:           row.ID,
		Email:        userdomain.Email(row.Email),
		PasswordHash: row.PasswordHash,
		Name:         row.Name,
		CreatedAt:    row.CreatedAt,
	}
}

func (r *UserRepository) Create(ctx context.Context, user *userdomain.User) error {
	result, err := r.db.ExecContext(
		ctx,
		`INSERT INTO users (email, password_hash, name, created_at) VALUES (?, ?, ?, ?)`,
		user.Email.String(),
		user.PasswordHash,
		user.Name,
		user.CreatedAt,
	)
	if isDuplicateKey(err) {
		return userdomain.ErrEmailTaken()
	}
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = id
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email userdomain.Email) (*userdomain.User, error) {
	return r.get(ctx, `SELECT id, email, password_hash, name, created_at FROM users WHERE email = ?`, email.String())
}

func (r *UserRepository) GetByID(ctx context.Context, userID int64) (*userdomain.User, error) {
	return r.get(ctx, `SELECT id, email, password_hash, name, created_at FROM users WHERE id = ?`, userID)
}

func (r *UserRepository) get(ctx context.Context, query string, arg any) (*userdomain.User, error) {
	var row userRow
	if err := r.db.GetContext(ctx, &row, query, arg); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	user := row.toDomain()
	return user, nil
}
