package positions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type ScanConfig struct {
	Interval   time.Duration
	StaleAfter time.Duration
	QueueSize  int
}

type Scanner struct {
	source staleTripSource
	cfg    ScanConfig
	queue  *queue
	logger *slog.Logger
	tracer trace.Tracer
	checks metric.Int64Counter
}

func NewScanner(
	source staleTripSource,
	cfg ScanConfig,
	logger *slog.Logger,
	tracer trace.Tracer,
	meter metric.Meter,
) (*Scanner, error) {
	switch {
	case cfg.Interval <= 0:
		return nil, errors.New("POSITION_CHECK_INTERVAL must be positive")
	case cfg.StaleAfter <= 0:
		return nil, errors.New("POSITION_STALE_AFTER must be positive")
	case cfg.QueueSize <= 0:
		return nil, errors.New("POSITION_QUEUE_SIZE must be positive")
	}
	checks, err := meter.Int64Counter("tripgo.stale_checks", metric.WithUnit("{check}"))
	if err != nil {
		return nil, fmt.Errorf("create stale checks metric: %w", err)
	}
	return &Scanner{
		source: source,
		cfg:    cfg,
		queue:  newQueue(cfg.QueueSize),
		logger: logger,
		tracer: tracer,
		checks: checks,
	}, nil
}

type systemTicker struct{ *time.Ticker }

func (t systemTicker) Ticks() <-chan time.Time { return t.C }

// Run owns the queue's write side and closes it when scanning stops. Call once.
func (s *Scanner) Run(ctx context.Context) {
	s.run(ctx, systemTicker{time.NewTicker(s.cfg.Interval)})
}

func (s *Scanner) run(ctx context.Context, ticker ticker) {
	defer close(s.queue.jobs)
	defer ticker.Stop()

	ticks := ticker.Ticks()
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ticks:
			if !ok || ctx.Err() != nil {
				return
			}
			s.scan(ctx)
		}
	}
}

func (s *Scanner) scan(ctx context.Context) {
	ctx, span := s.tracer.Start(ctx, "positions.scan_stale_trips")
	defer span.End()

	trips, err := s.source.ListStale(ctx, s.cfg.StaleAfter)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list stale trips")
		s.logger.WarnContext(ctx, "failed to list stale trips", "error", err)
		return
	}

	for _, trip := range trips {
		if ctx.Err() != nil {
			return
		}
		result := s.queue.tryEnqueue(job{trip: trip, scanSpan: span.SpanContext()})
		s.checks.Add(ctx, 1, metric.WithAttributes(attribute.String("result", string(result))))
		if result == dropped {
			s.logger.WarnContext(ctx, "position request queue is full", "trip_id", trip.TripID)
		}
	}
}
