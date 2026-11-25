package positionrepo

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/sashabeketov/avito-go-labworks/internal/infra/postgres/transaction"
)

type executor interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func executorFromContext(ctx context.Context, pool executor) executor {
	if tx, ok := transaction.FromContext(ctx); ok {
		return tx
	}
	return pool
}
