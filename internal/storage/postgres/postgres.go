package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sashabeketov/avito-go-labworks/internal/config"
)

func postgresConfig(cfg config.DBConfig) (*pgxpool.Config, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)

	if err != nil {
		return nil, fmt.Errorf("cannot parse database config: %w", err)
	}

	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime

	poolConfig.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	return poolConfig, nil
}

func CreateNewPool(
	ctx context.Context,
	cfg config.DBConfig) (*pgxpool.Pool, error) {
	poolConfig, err := postgresConfig(cfg)

	if err != nil {
		return nil, err
	}

	connectionPool, err := pgxpool.NewWithConfig(ctx, poolConfig)

	if err != nil {
		return nil, fmt.Errorf("cannot creat connections pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.PingTimeout)
	defer cancel()

	if err := connectionPool.Ping(pingCtx); err != nil {
		connectionPool.Close()
		return nil, fmt.Errorf("cannot ping PostgresSQL: %w", err)
	}

	return connectionPool, nil
}
