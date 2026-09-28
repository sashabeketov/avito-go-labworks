package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/sashabeketov/avito-go-labworks/internal/domain"
	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
)

func WriteParameterError(w http.ResponseWriter, r *http.Request, _ error) {
	writeInvalidRequest(w, r)
}

func writeInvalidRequest(w http.ResponseWriter, r *http.Request) {
	writeProblem(
		w, r,
		http.StatusBadRequest,
		"invalid_request",
		"invalid-request",
		"Invalid request",
		"Request validation failed",
	)
}

func writeTripError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidTrip):
		writeInvalidRequest(w, r)
	case errors.Is(err, domain.ErrTripNotFound):
		writeProblem(
			w, r,
			http.StatusNotFound,
			"trip_not_found",
			"trip-not-found",
			"Trip not found",
			"Trip was not found",
		)
	case errors.Is(err, domain.ErrDriverHasActiveTrip):
		writeProblem(
			w, r,
			http.StatusConflict,
			"driver_busy",
			"driver-busy",
			"Driver busy",
			"Driver already has an active trip",
		)
	case errors.Is(err, domain.ErrTripAlreadyCompleted):
		writeProblem(
			w, r,
			http.StatusConflict,
			"trip_completed",
			"trip-completed",
			"Trip completed",
			"Operation is not allowed for a completed trip",
		)
	default:
		slog.ErrorContext(r.Context(), "trip request failed", "error", err)
		writeProblem(
			w, r,
			http.StatusInternalServerError,
			"internal_error",
			"internal-error",
			"Internal Server Error",
			"Internal server error",
		)
	}
}

func writeProblem(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	code string,
	problemType string,
	title string,
	detail string,
) {
	instance := r.URL.Path
	problem := api.Problem{
		Type:     "https://tripgo.example/problems/" + problemType,
		Title:    title,
		Status:   int32(status),
		Detail:   &detail,
		Instance: &instance,
		Code:     code,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem)
}
