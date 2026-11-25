package domainerror

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
)

func Problem(err error) (int, api.Problem) {
	status := http.StatusInternalServerError
	code, kind, title, detail := "internal_error", "internal-error", "Internal Server Error", "Internal server error"
	switch {
	case errors.Is(err, trip.ErrInvalidTrip):
		status = http.StatusBadRequest
		code, kind, title, detail = "invalid_request", "invalid-request", "Invalid request", "Request validation failed"
	case errors.Is(err, trip.ErrTripNotFound):
		status = http.StatusNotFound
		code, kind, title, detail = "trip_not_found", "trip-not-found", "Trip not found", "Trip was not found"
	case errors.Is(err, trip.ErrDriverHasActiveTrip):
		status = http.StatusConflict
		code, kind, title, detail = "driver_busy", "driver-busy", "Driver busy", "Driver already has an active trip"
	case errors.Is(err, trip.ErrTripAlreadyCompleted):
		status = http.StatusConflict
		code, kind, title, detail = "trip_completed", "trip-completed", "Trip completed", "Operation is not allowed for a completed trip"
	}
	return status, api.Problem{
		Type:   "https://tripgo.example/problems/" + kind,
		Title:  title,
		Status: int32(status),
		Detail: &detail,
		Code:   code,
	}
}

func Write(w http.ResponseWriter, r *http.Request, err error, logger *slog.Logger) {
	status, body := Problem(err)
	span := trace.SpanFromContext(r.Context())
	span.RecordError(err)
	span.SetStatus(codes.Error, body.Code)
	span.SetAttributes(attribute.String("error.type", body.Code))
	if status == http.StatusInternalServerError {
		logger.ErrorContext(r.Context(), "trip request failed", "error", err)
	}

	instance := r.URL.Path
	body.Instance = &instance
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func WriteInvalidRequest(w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
	Write(w, r, trip.ErrInvalidTrip, logger)
}
