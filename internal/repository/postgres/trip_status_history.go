package postgres

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sashabeketov/avito-go-labworks/internal/domain"
)

type TripStatusHistoryRepository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewTripStatusHistoryRepository(
	pool *pgxpool.Pool,
	queryTimeout time.Duration,
) *TripStatusHistoryRepository {
	return &TripStatusHistoryRepository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (r *TripStatusHistoryRepository) Create(
	ctx context.Context,
	history domain.TripStatusHistory,
) error {
	query, args, err := sq.
		Insert("trip_status_history").
		Columns(
			"trip_id",
			"from_status",
			"to_status",
			"reason",
			"changed_at",
		).
		Values(
			history.TripID,
			history.FromStatus,
			history.ToStatus,
			history.Reason,
			history.ChangedAt,
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build create trip status history query: %w", err)
	}

	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	db := executorFromContext(queryCtx, r.pool)

	_, err = db.Exec(queryCtx, query, args...)
	if err != nil {
		return fmt.Errorf("create trip status history: %w", err)
	}

	return nil
}
