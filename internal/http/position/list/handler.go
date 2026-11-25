package list

import (
	"log/slog"
	"net/http"
	"uuid"

	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
	"github.com/sashabeketov/avito-go-labworks/internal/http/domainerror"
	"github.com/sashabeketov/avito-go-labworks/internal/http/position/response"
)

type Handler struct {
	useCase useCase
	logger  *slog.Logger
}

func New(useCase useCase, logger *slog.Logger) *Handler {
	return &Handler{useCase: useCase, logger: logger}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, id api.TripId) {
	positions, err := h.useCase.Execute(r.Context(), uuid.UUID(id))
	if err != nil {
		domainerror.Write(w, r, err, h.logger)
		return
	}
	result := make([]api.TripPosition, 0, len(positions))
	for _, position := range positions {
		result = append(result, response.Position(position))
	}
	response.Write(w, r, http.StatusOK, api.TripPositionsResponse{Positions: result}, h.logger)
}
