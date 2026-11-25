package create

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
	"github.com/sashabeketov/avito-go-labworks/internal/http/domainerror"
	"github.com/sashabeketov/avito-go-labworks/internal/http/trip/response"
	createuc "github.com/sashabeketov/avito-go-labworks/internal/usecase/trip/create"
)

type Handler struct {
	useCase useCase
	logger  *slog.Logger
}

func New(useCase useCase, logger *slog.Logger) *Handler {
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		domainerror.WriteInvalidRequest(w, r, h.logger)
		return
	}

	var raw json.RawMessage
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&raw); err != nil {
		domainerror.WriteInvalidRequest(w, r, h.logger)
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		domainerror.WriteInvalidRequest(w, r, h.logger)
		return
	}

	if !hasRequiredFields(raw) {
		domainerror.WriteInvalidRequest(w, r, h.logger)
		return
	}

	var req api.TripData
	strictDecoder := json.NewDecoder(bytes.NewReader(raw))
	strictDecoder.DisallowUnknownFields()
	if err := strictDecoder.Decode(&req); err != nil ||
		uuid.UUID(req.UserId) == uuid.Nil() ||
		uuid.UUID(req.DriverId) == uuid.Nil() || req.Price < 0 {
		domainerror.WriteInvalidRequest(w, r, h.logger)
		return
	}

	startPoint, err := vobject.NewCoordinates(req.StartPoint.Latitude, req.StartPoint.Longitude)
	if err != nil {
		domainerror.WriteInvalidRequest(w, r, h.logger)
		return
	}

	endPoint, err := vobject.NewCoordinates(req.EndPoint.Latitude, req.EndPoint.Longitude)
	if err != nil {
		domainerror.WriteInvalidRequest(w, r, h.logger)
		return
	}

	created, err := h.useCase.Execute(r.Context(), createuc.Input{
		UserID:     uuid.UUID(req.UserId),
		DriverID:   uuid.UUID(req.DriverId),
		StartPoint: startPoint,
		EndPoint:   endPoint,
		Price:      req.Price,
	})
	if err != nil {
		domainerror.Write(w, r, err, h.logger)
		return
	}

	response.Write(w, r, http.StatusCreated, created, h.logger)
}

func hasRequiredFields(raw json.RawMessage) bool {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return false
	}

	for _, name := range []string{"user_id", "driver_id", "start_point", "end_point", "price"} {
		if !present(fields, name) {
			return false
		}
	}

	for _, name := range []string{"start_point", "end_point"} {
		var coordinates map[string]json.RawMessage
		if err := json.Unmarshal(fields[name], &coordinates); err != nil || coordinates == nil {
			return false
		}
		if !present(coordinates, "latitude") || !present(coordinates, "longitude") {
			return false
		}
	}

	return true
}

func present(fields map[string]json.RawMessage, name string) bool {
	value, ok := fields[name]
	return ok && !bytes.Equal(bytes.TrimSpace(value), []byte("null"))
}
