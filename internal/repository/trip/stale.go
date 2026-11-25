package triprepo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

const listStaleTripsSQL = `
	SELECT id, driver_id
	FROM trips
	WHERE status = 'active'
	  AND COALESCE(last_position_at, started_at) < now() - $1::interval
	ORDER BY COALESCE(last_position_at, started_at), id
`

func (r *Repository) ListStale(ctx context.Context, staleAfter time.Duration) ([]model.StaleTrip, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	db := executorFromContext(queryCtx, r.pool)
	interval := pgtype.Interval{Microseconds: staleAfter.Microseconds(), Valid: true}
	rows, err := db.Query(queryCtx, listStaleTripsSQL, interval)
	if err != nil {
		return nil, fmt.Errorf("list stale trips: %w", err)
	}
	defer rows.Close()

	trips := make([]model.StaleTrip, 0)
	for rows.Next() {
		var trip model.StaleTrip
		if err := rows.Scan(&trip.TripID, &trip.DriverID); err != nil {
			return nil, fmt.Errorf("scan stale trip: %w", err)
		}
		trips = append(trips, trip)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read stale trips: %w", err)
	}
	return trips, nil
}
