package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"
)

func RegisterActiveTripsGauge(meter metric.Meter, pool *pgxpool.Pool, queryTimeout time.Duration) error {
	gauge, err := meter.Int64ObservableGauge("tripgo.trips.active", metric.WithUnit("{trip}"))
	if err != nil {
		return fmt.Errorf("create active trips gauge: %w", err)
	}

	_, err = meter.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
		queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
		defer cancel()

		var count int64
		if err := pool.QueryRow(queryCtx, "SELECT count(*) FROM trips WHERE status = 'active'").Scan(&count); err != nil {
			return fmt.Errorf("count active trips: %w", err)
		}
		observer.ObserveInt64(gauge, count)
		return nil
	}, gauge)
	if err != nil {
		return fmt.Errorf("register active trips gauge: %w", err)
	}
	return nil
}
