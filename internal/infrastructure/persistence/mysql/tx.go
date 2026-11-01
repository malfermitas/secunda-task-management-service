package mysql

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// withTx runs fn inside a transaction and commits only on success.
func withTx(ctx context.Context, db *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
