package triprepo

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/sashabeketov/avito-go-labworks/internal/infra/postgres/transaction"
)

// executor is the small part of pgxpool.Pool and pgx.Tx needed by this repository.
type executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func executorFromContext(ctx context.Context, pool executor) executor {
	if tx, ok := transaction.FromContext(ctx); ok {
		return tx
	}
	return pool
}
