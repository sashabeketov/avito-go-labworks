package positionrepo

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	tripdomain "github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type Repository struct {
	pool         executor
	queryTimeout time.Duration
}

func New(pool executor, queryTimeout time.Duration) *Repository {
	return &Repository{pool: pool, queryTimeout: queryTimeout}
}

// TouchActiveTrip locks the trip row until the surrounding transaction commits.
// The comparison is atomic, so an older coordinate cannot move the timestamp back.
func (r *Repository) TouchActiveTrip(ctx context.Context, tripID uuid.UUID, recordedAt, updatedAt time.Time) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	db := executorFromContext(queryCtx, r.pool)
	var id uuid.UUID
	err := db.QueryRow(queryCtx, `
		UPDATE trips
		SET last_position_at = GREATEST(COALESCE(last_position_at, $2), $2),
		    updated_at = $3
		WHERE id = $1 AND status = 'active'
		RETURNING id
	`, tripID, recordedAt, updatedAt).Scan(&id)
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("update last position time: %w", err)
	}

	var status string
	err = db.QueryRow(queryCtx, `SELECT status FROM trips WHERE id = $1`, tripID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return tripdomain.ErrTripNotFound
	}
	if err != nil {
		return fmt.Errorf("check trip after position update: %w", err)
	}
	if status == string(vobject.TripStatusCompleted) {
		return tripdomain.ErrTripAlreadyCompleted
	}
	return errors.New("active trip was not updated")
}

func (r *Repository) Create(ctx context.Context, position model.TripPosition) (model.TripPosition, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	db := executorFromContext(queryCtx, r.pool)
	created, err := scanPosition(db.QueryRow(queryCtx, `
		INSERT INTO trip_positions (trip_id, latitude, longitude, recorded_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, trip_id, latitude, longitude, recorded_at
	`, position.TripID, position.Coordinates.Latitude, position.Coordinates.Longitude, position.RecordedAt))
	if err != nil {
		return model.TripPosition{}, fmt.Errorf("insert trip position: %w", err)
	}
	return created, nil
}

func (r *Repository) ListByTrip(ctx context.Context, tripID uuid.UUID) ([]model.TripPosition, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	db := executorFromContext(queryCtx, r.pool)
	rows, err := db.Query(queryCtx, `
		SELECT id, trip_id, latitude, longitude, recorded_at
		FROM trip_positions
		WHERE trip_id = $1
		ORDER BY recorded_at, id
	`, tripID)
	if err != nil {
		return nil, fmt.Errorf("list trip positions: %w", err)
	}
	defer rows.Close()

	positions := make([]model.TripPosition, 0)
	for rows.Next() {
		position, scanErr := scanPosition(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan trip position: %w", scanErr)
		}
		positions = append(positions, position)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read trip positions: %w", err)
	}
	return positions, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPosition(row rowScanner) (model.TripPosition, error) {
	var position model.TripPosition
	var latitude, longitude float64
	if err := row.Scan(&position.ID, &position.TripID, &latitude, &longitude, &position.RecordedAt); err != nil {
		return model.TripPosition{}, err
	}
	coordinates, err := vobject.NewCoordinates(latitude, longitude)
	if err != nil {
		return model.TripPosition{}, fmt.Errorf("restore coordinates: %w", err)
	}
	position.Coordinates = coordinates
	return position, nil
}
