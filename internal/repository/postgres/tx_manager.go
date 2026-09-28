package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txContextKey struct{}

type TxManager struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewTxManager(
	pool *pgxpool.Pool,
	queryTimeout time.Duration,
) *TxManager {
	return &TxManager{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func transactionFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txContextKey{}).(pgx.Tx)
	return tx, ok
}

func (m *TxManager) rollback(ctx context.Context, tx pgx.Tx) error {
	rollbackCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		m.queryTimeout,
	)
	defer cancel()

	return tx.Rollback(rollbackCtx)
}

func (m *TxManager) Do(
	ctx context.Context,
	fn func(context.Context) error,
) error {
	if _, ok := transactionFromContext(ctx); ok {
		return fn(ctx)
	}

	beginCtx, cancel := context.WithTimeout(ctx, m.queryTimeout)
	tx, err := m.pool.BeginTx(beginCtx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	cancel()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	txCtx := context.WithValue(ctx, txContextKey{}, tx)

	defer func() {
		if p := recover(); p != nil {
			_ = m.rollback(ctx, tx)
			panic(p)
		}
	}()

	if err := fn(txCtx); err != nil {
		rollbackErr := m.rollback(ctx, tx)
		if rollbackErr != nil {
			return errors.Join(
				err,
				fmt.Errorf("rollback transaction: %w", rollbackErr),
			)
		}

		return err
	}

	commitCtx, cancel := context.WithTimeout(ctx, m.queryTimeout)
	err = tx.Commit(commitCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
