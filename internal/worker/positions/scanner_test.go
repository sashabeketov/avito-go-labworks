package positions

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/goleak"
	"go.uber.org/mock/gomock"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestScannerQueuesInOrderAndRecordsDuplicateAndDropped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctrl := gomock.NewController(t)
		source := NewMockstaleTripSource(ctrl)
		ticker, ticks := controlledTicker(ctrl)
		older := model.StaleTrip{TripID: uuid.New(), DriverID: uuid.New()}
		newer := model.StaleTrip{TripID: uuid.New(), DriverID: uuid.New()}
		overflow := model.StaleTrip{TripID: uuid.New(), DriverID: uuid.New()}
		const staleAfter = time.Minute
		gomock.InOrder(
			source.EXPECT().ListStale(gomock.Any(), staleAfter).Return([]model.StaleTrip{older, newer, overflow}, nil),
			source.EXPECT().ListStale(gomock.Any(), staleAfter).Return([]model.StaleTrip{older, newer, overflow}, nil),
			source.EXPECT().ListStale(gomock.Any(), staleAfter).Return([]model.StaleTrip{older}, nil),
		)

		reader := sdkmetric.NewManualReader()
		metrics := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		t.Cleanup(func() { require.NoError(t, metrics.Shutdown(context.Background())) })
		traces := sdktrace.NewTracerProvider()
		t.Cleanup(func() { require.NoError(t, traces.Shutdown(context.Background())) })
		var logs bytes.Buffer
		scanner, err := NewScanner(
			source, ScanConfig{Interval: 30 * time.Second, StaleAfter: staleAfter, QueueSize: 2},
			slog.New(slog.NewJSONHandler(&logs, nil)), traces.Tracer("test"), metrics.Meter("test"),
		)
		require.NoError(t, err)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan struct{})
		go func() {
			defer close(done)
			scanner.run(ctx, ticker)
		}()

		ticks <- time.Now()
		synctest.Wait()
		require.Len(t, scanner.queue.jobs, 2)
		first := <-scanner.queue.jobs
		require.Equal(t, older, first.trip)
		require.True(t, first.scanSpan.IsValid())

		ticks <- time.Now()
		synctest.Wait()
		require.Len(t, scanner.queue.jobs, 2)
		second := <-scanner.queue.jobs
		third := <-scanner.queue.jobs
		require.Equal(t, newer, second.trip)
		require.Equal(t, overflow, third.trip)
		scanner.queue.complete(first)
		scanner.queue.complete(second)
		scanner.queue.complete(third)

		ticks <- time.Now()
		synctest.Wait()
		require.Len(t, scanner.queue.jobs, 1)

		var data metricdata.ResourceMetrics
		require.NoError(t, reader.Collect(ctx, &data))
		require.Len(t, data.ScopeMetrics, 1)
		require.Len(t, data.ScopeMetrics[0].Metrics, 1)
		checks := data.ScopeMetrics[0].Metrics[0]
		require.Equal(t, "tripgo.stale_checks", checks.Name)
		datapoints, ok := checks.Data.(metricdata.Sum[int64])
		require.True(t, ok)
		totals := make(map[string]int64)
		for _, point := range datapoints.DataPoints {
			result, ok := point.Attributes.Value(attribute.Key("result"))
			require.True(t, ok)
			totals[result.AsString()] = point.Value
		}
		require.Equal(t, map[string]int64{string(queued): 4, string(duplicate): 2, string(dropped): 1}, totals)
		require.Contains(t, logs.String(), `"level":"WARN"`)
		require.Contains(t, logs.String(), overflow.TripID.String())

		cancel()
		synctest.Wait()
		requireClosed(t, done)
		require.Equal(t, older, (<-scanner.queue.jobs).trip)
		_, open := <-scanner.queue.jobs
		require.False(t, open)
	})
}

func TestScannerDoesNotOverlapQueriesAndStopsInFlight(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctrl := gomock.NewController(t)
		source := NewMockstaleTripSource(ctrl)
		ticker, ticks := controlledTicker(ctrl)
		release := make(chan struct{})
		calls := make(chan struct{}, 2)
		const staleAfter = time.Minute
		gomock.InOrder(
			source.EXPECT().ListStale(gomock.Any(), staleAfter).DoAndReturn(
				func(ctx context.Context, _ time.Duration) ([]model.StaleTrip, error) {
					calls <- struct{}{}
					select {
					case <-release:
						return nil, errors.New("temporary query error")
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				},
			),
			source.EXPECT().ListStale(gomock.Any(), staleAfter).DoAndReturn(
				func(ctx context.Context, _ time.Duration) ([]model.StaleTrip, error) {
					calls <- struct{}{}
					<-ctx.Done()
					return nil, ctx.Err()
				},
			),
		)
		scanner, err := NewScanner(
			source, ScanConfig{Interval: 30 * time.Second, StaleAfter: staleAfter, QueueSize: 1},
			slog.New(slog.NewTextHandler(io.Discard, nil)), tracenoop.NewTracerProvider().Tracer("test"),
			noop.NewMeterProvider().Meter("test"),
		)
		require.NoError(t, err)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan struct{})
		go func() {
			defer close(done)
			scanner.run(ctx, ticker)
		}()

		ticks <- time.Now()
		synctest.Wait()
		require.Len(t, calls, 1)
		ticks <- time.Now()
		synctest.Wait()
		require.Len(t, calls, 1)
		close(release)
		synctest.Wait()
		require.Len(t, calls, 2)

		cancel()
		synctest.Wait()
		requireClosed(t, done)
	})
}

func controlledTicker(ctrl *gomock.Controller) (*Mockticker, chan<- time.Time) {
	ticks := make(chan time.Time, 1)
	ticker := NewMockticker(ctrl)
	ticker.EXPECT().Ticks().Return((<-chan time.Time)(ticks))
	ticker.EXPECT().Stop()
	return ticker, ticks
}

func requireClosed(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	default:
		t.Fatal("scanner has not stopped")
	}
}
