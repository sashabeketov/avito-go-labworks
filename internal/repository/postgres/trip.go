package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sashabeketov/avito-go-labworks/internal/domain"
	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
	"github.com/sashabeketov/avito-go-labworks/internal/repository/postgres/mapper"
)

type TripRepository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewTripRepository(
	pool *pgxpool.Pool,
	queryTimeout time.Duration,
) *TripRepository {
	return &TripRepository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (r *TripRepository) Create(
	ctx context.Context,
	trip domain.Trip,
) error {
	query, args, err := sq.
		Insert("trips").
		Columns(
			"id",
			"user_id",
			"driver_id",
			"start_latitude",
			"start_longitude",
			"end_latitude",
			"end_longitude",
			"price",
			"status",
			"started_at",
			"finished_at",
		).
		Values(
			trip.ID,
			trip.UserID,
			trip.DriverID,
			trip.StartPoint.Latitude,
			trip.StartPoint.Longitude,
			trip.EndPoint.Latitude,
			trip.EndPoint.Longitude,
			trip.Price,
			trip.Status,
			trip.StartedAt,
			trip.FinishedAt,
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("build create trip query: %w", err)
	}

	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	db := executorFromContext(queryCtx, r.pool)

	_, err = db.Exec(queryCtx, query, args...)
	if err != nil {

		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_one_active_per_driver_idx" {
			return fmt.Errorf("create trip: %w: %w", domain.ErrDriverHasActiveTrip, err)
		}

		return fmt.Errorf("create trip: %w", err)
	}

	return nil
}

func (r *TripRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.Trip, error) {
	query, args, err := sq.
		Select(
			"id",
			"user_id",
			"driver_id",
			"start_latitude",
			"start_longitude",
			"end_latitude",
			"end_longitude",
			"price",
			"status",
			"started_at",
			"finished_at",
			"created_at",
			"updated_at",
		).
		From("trips").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return domain.Trip{}, fmt.Errorf(
			"build get trip query: %w",
			err,
		)
	}

	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	db := executorFromContext(queryCtx, r.pool)

	trip, err := mapper.MapTrip(
		db.QueryRow(queryCtx, query, args...),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Trip{}, domain.ErrTripNotFound
		}

		return domain.Trip{}, fmt.Errorf(
			"get trip: %w",
			err,
		)
	}

	return trip, nil
}

func (r *TripRepository) Finish(
	ctx context.Context,
	id uuid.UUID,
	finishedAt time.Time,
) (domain.Trip, error) {
	query, args, err := sq.
		Update("trips").
		Set("status", vobject.TripStatusCompleted).
		Set("finished_at", finishedAt).
		Set("updated_at", finishedAt).
		Where(sq.Eq{
			"id":     id,
			"status": vobject.TripStatusActive,
		}).
		Suffix(`
			RETURNING
				id,
				user_id,
				driver_id,
				start_latitude,
				start_longitude,
				end_latitude,
				end_longitude,
				price,
				status,
				started_at,
				finished_at,
				created_at,
				updated_at
		`).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return domain.Trip{}, fmt.Errorf(
			"build finish trip query: %w",
			err,
		)
	}

	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	db := executorFromContext(queryCtx, r.pool)

	trip, err := mapper.MapTrip(
		db.QueryRow(queryCtx, query, args...),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			existingTrip, lookupErr := r.GetByID(ctx, id)
			if errors.Is(lookupErr, domain.ErrTripNotFound) {
				return domain.Trip{}, domain.ErrTripNotFound
			}
			if lookupErr != nil {
				return domain.Trip{}, fmt.Errorf(
					"check trip after finish: %w",
					lookupErr,
				)
			}
			if existingTrip.Status == vobject.TripStatusCompleted {
				return domain.Trip{}, domain.ErrTripAlreadyCompleted
			}

			return domain.Trip{}, errors.New(
				"finish trip: active trip was not updated",
			)
		}

		return domain.Trip{}, fmt.Errorf(
			"finish trip: %w",
			err,
		)
	}

	return trip, nil
}
