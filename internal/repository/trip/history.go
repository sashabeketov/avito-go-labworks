package triprepo

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type HistoryRepository struct {
	pool         executor
	queryTimeout time.Duration
}

func NewHistoryRepository(
	pool executor,
	queryTimeout time.Duration,
) *HistoryRepository {
	return &HistoryRepository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (r *HistoryRepository) Create(
	ctx context.Context,
	history model.TripStatusHistory,
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
