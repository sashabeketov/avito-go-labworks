//go:build integration

package positionrepo_test

import (
	"context"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
	"github.com/sashabeketov/avito-go-labworks/internal/infra/postgres/transaction"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
	positionrepo "github.com/sashabeketov/avito-go-labworks/internal/repository/position"
	triprepo "github.com/sashabeketov/avito-go-labworks/internal/repository/trip"
	createuc "github.com/sashabeketov/avito-go-labworks/internal/usecase/position/create"
	listuc "github.com/sashabeketov/avito-go-labworks/internal/usecase/position/list"
)

const queryTimeout = 3 * time.Second

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type positionEvents struct{}

func (positionEvents) PositionSaved(context.Context, uuid.UUID) {}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	require.NoError(t, err)
	require.NoError(t, pool.Ping(ctx))
	t.Cleanup(pool.Close)
	return pool
}

func createTrip(t *testing.T, pool *pgxpool.Pool) model.Trip {
	t.Helper()
	initial := model.Trip{
		ID: uuid.New(), UserID: uuid.New(), DriverID: uuid.New(),
		StartPoint: vobject.Coordinates{Latitude: 55.75, Longitude: 37.62},
		EndPoint:   vobject.Coordinates{Latitude: 55.76, Longitude: 37.63},
		Price:      1450, Status: vobject.TripStatusActive, StartedAt: time.Now().Add(-time.Hour).UTC(),
	}
	ctx := context.Background()
	require.NoError(t, triprepo.NewTripRepository(pool, queryTimeout).Create(ctx, initial))
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), queryTimeout)
		defer cancel()
		_, err := pool.Exec(cleanupCtx, `DELETE FROM trips WHERE id = $1`, initial.ID)
		require.NoError(t, err)
	})
	return initial
}

func TestPositionRouteAndLateArrival(t *testing.T) {
	pool := testPool(t)
	initial := createTrip(t, pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	trips := triprepo.NewTripRepository(pool, queryTimeout)
	positions := positionrepo.New(pool, queryTimeout)
	create := createuc.New(positions, transaction.NewTxManager(pool, queryTimeout), fixedClock{now}, positionEvents{})
	list := listuc.New(trips, positions)

	newer, err := create.Execute(ctx, createuc.Input{TripID: initial.ID, Coordinates: vobject.Coordinates{Latitude: 59.93, Longitude: 30.33}, RecordedAt: now.Add(-10 * time.Minute)})
	require.NoError(t, err)
	older, err := create.Execute(ctx, createuc.Input{TripID: initial.ID, Coordinates: vobject.Coordinates{Latitude: 59.94, Longitude: 30.34}, RecordedAt: now.Add(-20 * time.Minute)})
	require.NoError(t, err)
	sameTime, err := create.Execute(ctx, createuc.Input{TripID: initial.ID, Coordinates: vobject.Coordinates{Latitude: 59.95, Longitude: 30.35}, RecordedAt: older.RecordedAt})
	require.NoError(t, err)

	gotTrip, err := trips.GetByID(ctx, initial.ID)
	require.NoError(t, err)
	require.NotNil(t, gotTrip.LastPositionAt)
	require.True(t, gotTrip.LastPositionAt.Equal(newer.RecordedAt))
	route, err := list.Execute(ctx, initial.ID)
	require.NoError(t, err)
	require.Len(t, route, 3)
	require.Equal(t, []int64{older.ID, sameTime.ID, newer.ID}, []int64{route[0].ID, route[1].ID, route[2].ID})

	_, err = trips.Finish(ctx, initial.ID, now)
	require.NoError(t, err)
	_, err = create.Execute(ctx, createuc.Input{TripID: initial.ID, Coordinates: vobject.Coordinates{Latitude: 1, Longitude: 2}, RecordedAt: now})
	require.ErrorIs(t, err, trip.ErrTripAlreadyCompleted)
	_, err = create.Execute(ctx, createuc.Input{TripID: uuid.New(), Coordinates: vobject.Coordinates{Latitude: 1, Longitude: 2}, RecordedAt: now})
	require.ErrorIs(t, err, trip.ErrTripNotFound)
}

func TestPositionInsertFailureRollsBackTripTimestamp(t *testing.T) {
	pool := testPool(t)
	initial := createTrip(t, pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	positions := positionrepo.New(pool, queryTimeout)
	tx := transaction.NewTxManager(pool, queryTimeout)
	err := tx.Do(ctx, func(txCtx context.Context) error {
		if err := positions.TouchActiveTrip(txCtx, initial.ID, now, now); err != nil {
			return err
		}
		_, err := positions.Create(txCtx, model.TripPosition{
			TripID: initial.ID, Coordinates: vobject.Coordinates{Latitude: 99, Longitude: 30}, RecordedAt: now,
		})
		return err
	})
	require.Error(t, err)

	var lastPositionAt *time.Time
	require.NoError(t, pool.QueryRow(ctx, `SELECT last_position_at FROM trips WHERE id = $1`, initial.ID).Scan(&lastPositionAt))
	require.Nil(t, lastPositionAt)
	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM trip_positions WHERE trip_id = $1`, initial.ID).Scan(&count))
	require.Zero(t, count)
}
