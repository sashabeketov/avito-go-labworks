package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

type queryTracer struct {
	*otelpgx.Tracer
	duration metric.Float64Histogram
}

type queryInfoKey struct{}

type queryInfo struct {
	started    time.Time
	operation  string
	collection string
}

func newQueryTracer(tp trace.TracerProvider, mp metric.MeterProvider) (*queryTracer, error) {
	duration, err := mp.Meter("trip-service/postgres").Float64Histogram(
		"db.client.operation.duration",
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10),
	)
	if err != nil {
		return nil, fmt.Errorf("create database duration metric: %w", err)
	}
	return &queryTracer{
		Tracer: otelpgx.NewTracer(
			otelpgx.WithTracerProvider(tp),
			otelpgx.WithMeterProvider(noop.NewMeterProvider()),
			otelpgx.WithDisableSQLStatementInAttributes(),
		),
		duration: duration,
	}, nil
}

func (t *queryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	operation := otelpgx.SQLOperationName(ctx, data.SQL)
	collection := collectionName(data.SQL)
	ctx = t.Tracer.TraceQueryStart(ctx, conn, data)
	trace.SpanFromContext(ctx).SetAttributes(
		attribute.String("db.operation.name", operation),
		attribute.String("db.collection.name", collection),
	)
	return context.WithValue(ctx, queryInfoKey{}, queryInfo{
		started:    time.Now(),
		operation:  operation,
		collection: collection,
	})
}

func (t *queryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	if info, ok := ctx.Value(queryInfoKey{}).(queryInfo); ok {
		t.duration.Record(ctx, time.Since(info.started).Seconds(), metric.WithAttributes(
			attribute.String("db.operation.name", info.operation),
			attribute.String("db.collection.name", info.collection),
			attribute.String("error.type", errorType(data.Err)),
		))
	}
	if data.Err != nil {
		trace.SpanFromContext(ctx).SetAttributes(attribute.String("error.type", errorType(data.Err)))
	}
	t.Tracer.TraceQueryEnd(ctx, conn, data)
}

func collectionName(sql string) string {
	query := strings.ToLower(sql)
	switch {
	case strings.Contains(query, "trip_status_history"):
		return "trip_status_history"
	case strings.Contains(query, "trips"):
		return "trips"
	default:
		return "unknown"
	}
}

func errorType(err error) string {
	if err == nil {
		return "none"
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline_exceeded"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	return "database_error"
}
