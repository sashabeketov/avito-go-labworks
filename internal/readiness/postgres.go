package readiness

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresChecker struct {
	pool        *pgxpool.Pool
	pingTimeout time.Duration
}

func NewPostgresChecker(
	pool *pgxpool.Pool,
	pingTimeout time.Duration) *PostgresChecker {
	return &PostgresChecker{
		pool:        pool,
		pingTimeout: pingTimeout,
	}
}

func (ch *PostgresChecker) Name() string {
	return "postgres"
}

func (ch *PostgresChecker) Check(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, ch.pingTimeout)
	defer cancel()

	err := ch.pool.Ping(pingCtx)
	if err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	return nil
}
