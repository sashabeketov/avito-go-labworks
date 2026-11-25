package get

import (
	"log/slog"
	"net/http"
	"uuid"

	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
	"github.com/sashabeketov/avito-go-labworks/internal/http/domainerror"
	"github.com/sashabeketov/avito-go-labworks/internal/http/trip/response"
)

type Handler struct {
	useCase useCase
	logger  *slog.Logger
}

func New(useCase useCase, logger *slog.Logger) *Handler {
	return &Handler{useCase: useCase, logger: logger}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, id api.TripId) {
	result, err := h.useCase.Execute(r.Context(), uuid.UUID(id))
	if err != nil {
		domainerror.Write(w, r, err, h.logger)
		return
	}
	response.Write(w, r, http.StatusOK, result, h.logger)
}
