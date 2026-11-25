package positions

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

func TestPoolDrainsWithFixedWorkersAndSkipsCancelledJob(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctrl := gomock.NewController(t)
		source := NewMockstaleTripSource(ctrl)
		push := NewMockpushGateway(ctrl)
		ticker, ticks := controlledTicker(ctrl)
		pool := newTestPool(t, source, push, 2)
		trips := testTrips(5)
		source.EXPECT().ListStale(gomock.Any(), time.Minute).Return(trips, nil)

		attempts := make(chan model.StaleTrip, len(trips))
		release := make(chan struct{})
		for i, trip := range trips[:4] {
			push.EXPECT().RequestPosition(gomock.Any(), trip).DoAndReturn(func(ctx context.Context, trip model.StaleTrip) error {
				attempts <- trip
				select {
				case <-release:
					if i == 0 {
						return errors.New("Push Service unavailable")
					}
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			})
		}

		appCtx, cancelApp := context.WithCancel(t.Context())
		defer cancelApp()
		pool.start(appCtx, func(ctx context.Context) { pool.scanner.run(ctx, ticker) })
		ticks <- time.Now()
		synctest.Wait()
		require.Len(t, attempts, 2)
		require.EqualValues(t, 2, pool.active.Load())
		pool.PositionSaved(t.Context(), trips[4].TripID)
		cancelApp()

		result := make(chan error, 1)
		go func() { result <- pool.Shutdown(t.Context()) }()
		synctest.Wait()
		require.Empty(t, result)
		require.EqualValues(t, 2, pool.active.Load())
		close(release)
		require.NoError(t, <-result)
		require.Len(t, attempts, 4)
		require.Zero(t, pool.active.Load())
		requireClosed(t, pool.done)
	})
}

func TestPoolCancelsRunningRequestAtShutdownDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctrl := gomock.NewController(t)
		source := NewMockstaleTripSource(ctrl)
		push := NewMockpushGateway(ctrl)
		ticker, ticks := controlledTicker(ctrl)
		pool := newTestPool(t, source, push, 1)
		trips := testTrips(2)
		source.EXPECT().ListStale(gomock.Any(), time.Minute).Return(trips, nil)
		push.EXPECT().RequestPosition(gomock.Any(), trips[0]).DoAndReturn(func(ctx context.Context, _ model.StaleTrip) error {
			<-ctx.Done()
			return ctx.Err()
		})

		pool.start(t.Context(), func(ctx context.Context) { pool.scanner.run(ctx, ticker) })
		ticks <- time.Now()
		synctest.Wait()
		require.EqualValues(t, 1, pool.active.Load())
		shutdownCtx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
		defer cancel()
		require.ErrorIs(t, pool.Shutdown(shutdownCtx), context.DeadlineExceeded)
		requireClosed(t, pool.done)
	})
}

func TestCancelledJobCannotClaimRequeuedTrip(t *testing.T) {
	q := newQueue(2)
	trip := model.StaleTrip{TripID: uuid.New()}
	require.Equal(t, queued, q.tryEnqueue(job{trip: trip}))
	q.cancelWaiting(trip.TripID)
	require.Equal(t, queued, q.tryEnqueue(job{trip: trip}))

	old := <-q.jobs
	require.False(t, q.claim(old))
	q.complete(old)
	current := <-q.jobs
	require.True(t, q.claim(current))
	q.cancelWaiting(trip.TripID)
	require.Equal(t, duplicate, q.tryEnqueue(job{trip: trip}))
	q.complete(current)
	require.Equal(t, queued, q.tryEnqueue(job{trip: trip}))
}

func newTestPool(t *testing.T, source staleTripSource, push pushGateway, workers int) *Pool {
	t.Helper()
	meter := noop.NewMeterProvider().Meter("test")
	scanner, err := NewScanner(source, ScanConfig{Interval: time.Second, StaleAfter: time.Minute, QueueSize: 10},
		slog.New(slog.NewTextHandler(io.Discard, nil)), tracenoop.NewTracerProvider().Tracer("test"), meter)
	require.NoError(t, err)
	pool, err := NewPool(scanner, push, workers, time.Second, meter)
	require.NoError(t, err)
	t.Cleanup(func() {
		if pool.stopScan != nil {
			pool.stopScan()
			pool.stopWorkers()
			<-pool.done
		}
	})
	return pool
}

func testTrips(count int) []model.StaleTrip {
	trips := make([]model.StaleTrip, 0, count)
	for range count {
		trips = append(trips, model.StaleTrip{TripID: uuid.New(), DriverID: uuid.New()})
	}
	return trips
}
