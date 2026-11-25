package health

import (
	"encoding/json"
	"net/http"

	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
)

type Handler struct{ readiness readinessChecker }

func New(readiness readinessChecker) *Handler { return &Handler{readiness: readiness} }

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: api.Ok})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	status, bodyStatus := http.StatusOK, api.Ok
	if h.readiness.CheckAll(r.Context()) != nil {
		status, bodyStatus = http.StatusServiceUnavailable, api.Unavailable
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: bodyStatus})
}
