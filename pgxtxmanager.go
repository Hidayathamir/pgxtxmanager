// Package pgxtxmanager provides utilities for managing pgx sql transactions.
package pgxtxmanager

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

// DBTx defines the interface able to do transactions.
type DBTx interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type ctxKey string

// ctxKey represents the context key used for storing the transaction.
var CtxKey = ctxKey("pgxtxmanager-sql-transaction")

// SQLTransaction executes a function within a PostgreSQL transaction. It begins a new transaction if none exists in the context,
// otherwise, it uses the existing one. If an error occurs during the execution of the provided function, it rolls back the transaction.
// If no external transaction exists, it commits the transaction upon successful execution of the function.
func SQLTransaction(ctx context.Context, dbTx DBTx, fn func(context.Context) error) error {
	tx, isHasExternalTransaction := ctx.Value(CtxKey).(pgx.Tx)

	if !isHasExternalTransaction {
		var err error
		tx, err = dbTx.Begin(ctx)
		if err != nil {
			return fmt.Errorf("DBTx.Begin: %w", err)
		}
		ctx = context.WithValue(ctx, CtxKey, tx)
	}

	err := fn(ctx)

	if !isHasExternalTransaction {
		if err != nil {
			errRollback := tx.Rollback(ctx)
			if errRollback != nil {
				slog.Warn("pgx.Tx.Rollback: %v", errRollback)
			}
			return err
		}
		errCommit := tx.Commit(ctx)
		if errCommit != nil {
			return fmt.Errorf("pgx.Tx.Commit: %w", errCommit)
		}
	}

	return err
}

// GetTxFromContext retrieves the PostgreSQL transaction from the context, if available.
func GetTxFromContext(ctx context.Context) (pgx.Tx, bool) { //nolint:ireturn
	tx, ok := ctx.Value(CtxKey).(pgx.Tx)
	return tx, ok
}
