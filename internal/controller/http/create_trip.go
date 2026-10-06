package http

import (
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

	var req createTripRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err = decoder.Decode(&req)
	if err != nil {
		writeInvalidRequest(w, r)
		return
	}

	err = validate.Struct(req)
	if err != nil {
		writeInvalidRequest(w, r)
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeInvalidRequest(w, r)
		return
	}

	startPoint, err := vobject.NewCoordinates(
		req.StartPoint.Latitude,
		req.StartPoint.Longitude)

	if err != nil {
		writeInvalidRequest(w, r)
		return
	}

	endPoint, err := vobject.NewCoordinates(
		req.EndPoint.Latitude,
		req.EndPoint.Longitude)

	if err != nil {
		writeInvalidRequest(w, r)
		return
	}

	input := service.CreateTripInput{
		UserID:     req.UserID,
		DriverID:   req.DriverID,
		StartPoint: startPoint,
		EndPoint:   endPoint,
		Price:      req.Price,
	}

	trip, err := h.tripService.Create(r.Context(), input)
	if err != nil {
		writeTripError(w, r, err)
		return
	}

	writeTrip(w, r, http.StatusCreated, trip)
}

type createTripRequest struct {
	UserID     uuid.UUID   `json:"user_id" validate:"required,uuid"`
	DriverID   uuid.UUID   `json:"driver_id" validate:"required,uuid"`
	StartPoint coordinates `json:"start_point" validate:"required"`
	EndPoint   coordinates `json:"end_point" validate:"required"`
	Price      int64       `json:"price" validate:"required,gte=0"`
}

type coordinates struct {
	Latitude  float64 `json:"latitude" validate:"required,gte=-90,lte=90"`
	Longitude float64 `json:"longitude" validate:"required,gte=-180,lte=180"`
}
