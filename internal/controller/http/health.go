package http

import (
	"encoding/json"
	"net/http"

	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
)

// Ready надо будет реализовать вывод или логирование ошибок проверок узлов
func (h *Handler) Ready(
	w http.ResponseWriter,
	r *http.Request) {
	err := h.readiness.CheckAll(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)

		_ = json.NewEncoder(w).Encode(api.HealthResponse{
			Status: api.Unavailable,
		})

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(api.HealthResponse{
		Status: api.Ok,
	})
}

func (h *Handler) Health(
	w http.ResponseWriter,
	r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(api.HealthResponse{
		Status: api.Ok,
	})
}
