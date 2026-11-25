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
	"github.com/sashabeketov/avito-go-labworks/internal/http/position/response"
	createuc "github.com/sashabeketov/avito-go-labworks/internal/usecase/position/create"
)

type Handler struct {
	useCase useCase
	logger  *slog.Logger
}

func New(useCase useCase, logger *slog.Logger) *Handler {
	return &Handler{useCase: useCase, logger: logger}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, id api.TripId) {
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
	if err := decoder.Decode(&struct{}{}); err != io.EOF || !hasRequiredFields(raw) {
		domainerror.WriteInvalidRequest(w, r, h.logger)
		return
	}

	var request api.PositionData
	strictDecoder := json.NewDecoder(bytes.NewReader(raw))
	strictDecoder.DisallowUnknownFields()
	if err := strictDecoder.Decode(&request); err != nil {
		domainerror.WriteInvalidRequest(w, r, h.logger)
		return
	}

	saved, err := h.useCase.Execute(r.Context(), createuc.Input{
		TripID: uuid.UUID(id),
		Coordinates: vobject.Coordinates{
			Latitude:  request.Coordinates.Latitude,
			Longitude: request.Coordinates.Longitude,
		},
		RecordedAt: request.RecordedAt,
	})
	if err != nil {
		domainerror.Write(w, r, err, h.logger)
		return
	}
	response.Write(w, r, http.StatusCreated, response.Position(saved), h.logger)
}

func hasRequiredFields(raw json.RawMessage) bool {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil ||
		!present(fields, "coordinates") || !present(fields, "recorded_at") {
		return false
	}
	var coordinates map[string]json.RawMessage
	if err := json.Unmarshal(fields["coordinates"], &coordinates); err != nil || coordinates == nil {
		return false
	}
	return present(coordinates, "latitude") && present(coordinates, "longitude")
}

func present(fields map[string]json.RawMessage, name string) bool {
	value, ok := fields[name]
	return ok && !bytes.Equal(bytes.TrimSpace(value), []byte("null"))
}
