package response

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
	"github.com/sashabeketov/avito-go-labworks/internal/http/domainerror"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

func Write(w http.ResponseWriter, r *http.Request, status int, trip model.Trip, logger *slog.Logger) {
	response := api.Trip{
		Id:         api.TripId(trip.ID),
		UserId:     api.TripId(trip.UserID),
		DriverId:   api.TripId(trip.DriverID),
		StartPoint: api.Coordinates{Latitude: trip.StartPoint.Latitude, Longitude: trip.StartPoint.Longitude},
		EndPoint:   api.Coordinates{Latitude: trip.EndPoint.Latitude, Longitude: trip.EndPoint.Longitude},
		Price:      int64(trip.Price),
		Status:     api.TripStatus(trip.Status),
		StartedAt:  trip.StartedAt,
		FinishedAt: trip.FinishedAt,
	}
	body, err := json.Marshal(response)
	if err != nil {
		domainerror.Write(w, r, fmt.Errorf("encode trip response: %w", err), logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if status == http.StatusCreated {
		w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	}
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}
