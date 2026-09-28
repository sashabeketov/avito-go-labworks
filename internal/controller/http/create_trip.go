package http

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
	"github.com/sashabeketov/avito-go-labworks/internal/service"
)

func (h *Handler) CreateTrip(
	w http.ResponseWriter,
	r *http.Request,
	_ api.CreateTripParams,
) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeInvalidRequest(w, r)
		return
	}

	var raw json.RawMessage
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&raw); err != nil {
		writeInvalidRequest(w, r)
		return
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		writeInvalidRequest(w, r)
		return
	}

	fields, ok := requiredFields(raw, "user_id", "driver_id", "start_point", "end_point", "price")
	if !ok {
		writeInvalidRequest(w, r)
		return
	}

	if _, ok := requiredFields(fields["start_point"], "latitude", "longitude"); !ok {
		writeInvalidRequest(w, r)
		return
	}
	if _, ok := requiredFields(fields["end_point"], "latitude", "longitude"); !ok {
		writeInvalidRequest(w, r)
		return
	}

	var body api.TripData
	strictDecoder := json.NewDecoder(bytes.NewReader(raw))
	strictDecoder.DisallowUnknownFields()
	if err := strictDecoder.Decode(&body); err != nil {
		writeInvalidRequest(w, r)
		return
	}

	input := service.CreateTripInput{
		UserID:     uuid.UUID(body.UserId),
		DriverID:   uuid.UUID(body.DriverId),
		StartPoint: vobject.Coordinates{Latitude: body.StartPoint.Latitude, Longitude: body.StartPoint.Longitude},
		EndPoint:   vobject.Coordinates{Latitude: body.EndPoint.Latitude, Longitude: body.EndPoint.Longitude},
		Price:      body.Price,
	}

	trip, err := h.tripService.Create(r.Context(), input)
	if err != nil {
		writeTripError(w, r, err)
		return
	}

	writeTrip(w, r, http.StatusCreated, trip)
}

func requiredFields(raw json.RawMessage, names ...string) (map[string]json.RawMessage, bool) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, false
	}

	for _, name := range names {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, false
		}
	}

	return fields, true
}
