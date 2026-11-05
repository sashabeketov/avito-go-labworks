package http

import (
	"net/http"
	"uuid"

	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
)

func (h *Handler) GetTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	trip, err := h.tripService.GetByID(r.Context(), uuid.UUID(tripID))
	if err != nil {
		writeTripError(w, r, err)
		return
	}

	writeTrip(w, r, http.StatusOK, trip)
}
