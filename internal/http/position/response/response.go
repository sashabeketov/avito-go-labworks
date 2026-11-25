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

func Position(position model.TripPosition) api.TripPosition {
	return api.TripPosition{
		Id:     position.ID,
		TripId: api.TripId(position.TripID),
		Coordinates: api.Coordinates{
			Latitude:  position.Coordinates.Latitude,
			Longitude: position.Coordinates.Longitude,
		},
		RecordedAt: position.RecordedAt,
	}
}

func Write(w http.ResponseWriter, r *http.Request, status int, value any, logger *slog.Logger) {
	body, err := json.Marshal(value)
	if err != nil {
		domainerror.Write(w, r, fmt.Errorf("encode position response: %w", err), logger)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}
