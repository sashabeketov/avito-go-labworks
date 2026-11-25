//go:build integration

package triprepo_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
	triprepo "github.com/sashabeketov/avito-go-labworks/internal/repository/trip"
)

func TestListStaleTrips(t *testing.T) {
	pool := testPool(t)
	ctx := t.Context()
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
		defer cancel()
		require.NoError(t, tx.Rollback(ctx))
	})

	// PostgreSQL uses one value of now() throughout the transaction.
	var now time.Time
	require.NoError(t, tx.QueryRow(ctx, "SELECT now()").Scan(&now))
	const staleAfter = 2 * time.Minute
	repo := triprepo.NewTripRepository(tx, queryTimeout)
	fixtures := []struct {
		startedAt      time.Time
		lastPositionAt *time.Time
		completed      bool
	}{
		{startedAt: now.Add(-15 * time.Minute), lastPositionAt: new(now.Add(-3 * time.Minute))},
		{startedAt: now.Add(-10 * time.Minute)},
		{startedAt: now.Add(-15 * time.Minute), lastPositionAt: new(now.Add(-time.Minute))},
		{startedAt: now.Add(-time.Minute)},
		{startedAt: now.Add(-staleAfter)},
		{startedAt: now.Add(-20 * time.Minute), completed: true},
	}

	seeded := make(map[uuid.UUID]struct{}, len(fixtures))
	ids := make([]model.StaleTrip, 0, len(fixtures))
	for _, fixture := range fixtures {
		trip := newTrip()
		trip.StartedAt = fixture.startedAt
		require.NoError(t, repo.Create(ctx, trip))
		if fixture.lastPositionAt != nil {
			_, err := tx.Exec(ctx, "UPDATE trips SET last_position_at = $2 WHERE id = $1", trip.ID, fixture.lastPositionAt)
			require.NoError(t, err)
		}
		if fixture.completed {
			_, err := repo.Finish(ctx, trip.ID, now)
			require.NoError(t, err)
		}
		seeded[trip.ID] = struct{}{}
		ids = append(ids, model.StaleTrip{TripID: trip.ID, DriverID: trip.DriverID})
	}

	trips, err := repo.ListStale(ctx, staleAfter)
	require.NoError(t, err)
	var ownTrips []model.StaleTrip
	for _, trip := range trips {
		if _, ok := seeded[trip.TripID]; ok {
			ownTrips = append(ownTrips, trip)
		}
	}
	require.Equal(t, []model.StaleTrip{ids[1], ids[0]}, ownTrips)
}
