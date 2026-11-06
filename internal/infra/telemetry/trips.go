package telemetry

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/metric"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type TripEvents struct {
	logger    *slog.Logger
	created   metric.Int64Counter
	completed metric.Int64Counter
}

func NewTripEvents(meter metric.Meter, logger *slog.Logger) (*TripEvents, error) {
	created, err := meter.Int64Counter("tripgo.trips.created", metric.WithUnit("{trip}"))
	if err != nil {
		return nil, fmt.Errorf("create trips created metric: %w", err)
	}
	completed, err := meter.Int64Counter("tripgo.trips.completed", metric.WithUnit("{trip}"))
	if err != nil {
		return nil, fmt.Errorf("create trips completed metric: %w", err)
	}
	return &TripEvents{logger: logger, created: created, completed: completed}, nil
}

func (e *TripEvents) Created(ctx context.Context, trip model.Trip) {
	e.created.Add(ctx, 1)
	e.logger.InfoContext(ctx, "trip created", "trip_id", trip.ID.String())
}

func (e *TripEvents) Completed(ctx context.Context, trip model.Trip) {
	e.completed.Add(ctx, 1)
	e.logger.InfoContext(ctx, "trip completed", "trip_id", trip.ID.String())
}
