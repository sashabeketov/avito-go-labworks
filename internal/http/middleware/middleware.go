package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/sashabeketov/avito-go-labworks/internal/http/domainerror"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if value := recover(); value != nil {
					logger.ErrorContext(r.Context(), "panic recovered", "panic", fmt.Sprint(value), "stack", string(debug.Stack()))
					domainerror.Write(w, r, errors.New("panic recovered"), logger)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func Tracing(
	tracerProvider trace.TracerProvider,
	meterProvider metric.MeterProvider,
	propagator propagation.TextMapPropagator,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				span := trace.SpanFromContext(r.Context())
				if route := routePattern(r); route != "" {
					r.Pattern = route
					span.SetName(r.Method + " " + route)
					span.SetAttributes(attribute.String("http.route", route))
				}
				if value := recover(); value != nil {
					span.RecordError(fmt.Errorf("panic: %v", value))
					span.SetStatus(codes.Error, "panic")
					span.SetAttributes(attribute.String("error.type", "panic"))
					panic(value)
				}
			}()
			next.ServeHTTP(w, r)
		})
		return otelhttp.NewHandler(wrapped, "HTTP",
			otelhttp.WithTracerProvider(tracerProvider),
			otelhttp.WithMeterProvider(meterProvider),
			otelhttp.WithPropagators(propagator),
		)
	}
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			status := http.StatusOK
			writer := httpsnoop.Wrap(w, httpsnoop.Hooks{
				WriteHeader: func(writeHeader httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
					return func(code int) {
						status = code
						writeHeader(code)
					}
				},
			})
			defer func() {
				value := recover()
				if value != nil {
					status = http.StatusInternalServerError
				}
				attrs := []any{
					"method", r.Method,
					"route", routePattern(r),
					"status", status,
					"duration_ms", time.Since(started).Milliseconds(),
				}
				if status >= http.StatusInternalServerError {
					logger.ErrorContext(r.Context(), "request handled", attrs...)
				} else {
					logger.InfoContext(r.Context(), "request handled", attrs...)
				}
				if value != nil {
					panic(value)
				}
			}()
			next.ServeHTTP(writer, r)
		})
	}
}

func Metrics(meter metric.Meter, router *chi.Mux) (func(http.Handler) http.Handler, error) {
	active, err := meter.Int64UpDownCounter("http.server.active_requests", metric.WithUnit("{request}"))
	if err != nil {
		return nil, fmt.Errorf("create active requests metric: %w", err)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := matchPattern(router, r)
			attrs := metric.WithAttributes(
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", route),
			)
			active.Add(r.Context(), 1, attrs)
			defer active.Add(r.Context(), -1, attrs)
			next.ServeHTTP(w, r)
		})
	}, nil
}

func routePattern(r *http.Request) string {
	if route := chi.RouteContext(r.Context()); route != nil {
		if pattern := route.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return "unmatched"
}

func matchPattern(router *chi.Mux, r *http.Request) string {
	ctx := chi.NewRouteContext()
	if router.Match(ctx, r.Method, r.URL.Path) {
		return ctx.RoutePattern()
	}
	return "unmatched"
}
