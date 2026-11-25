//go:build integration

package triprepo_test

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
	triprepo "github.com/sashabeketov/avito-go-labworks/internal/repository/trip"
)

const queryTimeout = 3 * time.Second

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

func cleanupTrip(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
		defer cancel()
		_, err := pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", id)
		require.NoError(t, err)
	})
}

func newTrip() model.Trip {
	return model.Trip{
		ID: uuid.New(), UserID: uuid.New(), DriverID: uuid.New(),
		StartPoint: vobject.Coordinates{Latitude: 55.75, Longitude: 37.62},
		EndPoint:   vobject.Coordinates{Latitude: 55.76, Longitude: 37.63},
		Price:      1450, Status: vobject.TripStatusActive, StartedAt: time.Now().UTC(),
	}
}

func TestRepositoryLifecycle(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := triprepo.NewTripRepository(pool, queryTimeout)
	initial := newTrip()
	cleanupTrip(t, pool, initial.ID)
	require.NoError(t, repo.Create(ctx, initial))

	got, err := repo.GetByID(ctx, initial.ID)
	require.NoError(t, err)
	require.Equal(t, initial.ID, got.ID)
	require.Equal(t, vobject.TripStatusActive, got.Status)

	busy := newTrip()
	busy.DriverID = initial.DriverID
	cleanupTrip(t, pool, busy.ID)
	require.ErrorIs(t, repo.Create(ctx, busy), trip.ErrDriverHasActiveTrip)

	finished, err := repo.Finish(ctx, initial.ID, time.Now().UTC())
	require.NoError(t, err)
	require.Equal(t, vobject.TripStatusCompleted, finished.Status)
	require.NotNil(t, finished.FinishedAt)
	_, err = repo.Finish(ctx, initial.ID, time.Now().UTC())
	require.ErrorIs(t, err, trip.ErrTripAlreadyCompleted)
	_, err = repo.GetByID(ctx, uuid.New())
	require.ErrorIs(t, err, trip.ErrTripNotFound)
}

func TestTransactionRollsBackTripAndHistory(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	initial := newTrip()
	cleanupTrip(t, pool, initial.ID)
	repo := triprepo.NewTripRepository(pool, queryTimeout)
	history := triprepo.NewHistoryRepository(pool, queryTimeout)
	tx := transaction.NewTxManager(pool, queryTimeout)

	err := tx.Do(ctx, func(txCtx context.Context) error {
		if err := repo.Create(txCtx, initial); err != nil {
			return err
		}
		return history.Create(txCtx, model.TripStatusHistory{
			TripID: initial.ID, ToStatus: "invalid", ChangedAt: time.Now().UTC(),
		})
	})
	require.Error(t, err)

	var tripCount, historyCount int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM trips WHERE id = $1", initial.ID).Scan(&tripCount))
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM trip_status_history WHERE trip_id = $1", initial.ID).Scan(&historyCount))
	require.Zero(t, tripCount)
	require.Zero(t, historyCount)
}
