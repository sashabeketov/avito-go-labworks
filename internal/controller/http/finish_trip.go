package http

import (
	"net/http"
	"uuid"

	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
)

func (h *Handler) FinishTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	trip, err := h.tripService.Finish(r.Context(), uuid.UUID(tripID))
	if err != nil {
		writeTripError(w, r, err)
		return
	}

	writeTrip(w, r, http.StatusOK, trip)
}
