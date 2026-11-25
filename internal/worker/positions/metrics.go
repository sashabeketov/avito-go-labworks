package positions

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

type poolMetrics struct {
	saved    metric.Int64Counter
	requests metric.Int64Counter
	duration metric.Float64Histogram
}

func newPoolMetrics(meter metric.Meter, queued, active func() int64) (poolMetrics, error) {
	saved, err := meter.Int64Counter("tripgo.positions.saved", metric.WithUnit("{position}"))
	if err != nil {
		return poolMetrics{}, fmt.Errorf("create saved positions counter: %w", err)
	}
	requests, err := meter.Int64Counter("tripgo.position_requests", metric.WithUnit("{request}"))
	if err != nil {
		return poolMetrics{}, fmt.Errorf("create position requests counter: %w", err)
	}
	duration, err := meter.Float64Histogram("tripgo.position_requests.duration", metric.WithUnit("s"))
	if err != nil {
		return poolMetrics{}, fmt.Errorf("create position request duration histogram: %w", err)
	}
	queueSize, err := meter.Int64ObservableGauge("tripgo.position_requests.queue.size", metric.WithUnit("{task}"))
	if err != nil {
		return poolMetrics{}, fmt.Errorf("create queue size gauge: %w", err)
	}
	activeWorkers, err := meter.Int64ObservableGauge("tripgo.position_requests.workers.active", metric.WithUnit("{worker}"))
	if err != nil {
		return poolMetrics{}, fmt.Errorf("create active workers gauge: %w", err)
	}
	_, err = meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
		observer.ObserveInt64(queueSize, queued())
		observer.ObserveInt64(activeWorkers, active())
		return nil
	}, queueSize, activeWorkers)
	if err != nil {
		return poolMetrics{}, fmt.Errorf("register pool gauges: %w", err)
	}
	return poolMetrics{saved: saved, requests: requests, duration: duration}, nil
}
