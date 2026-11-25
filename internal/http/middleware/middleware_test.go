package middleware_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
	"github.com/sashabeketov/avito-go-labworks/internal/http/middleware"
)

func TestLoggingUsesRouteTemplateAndResponseLevel(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		level  string
	}{
		{name: "client error", status: http.StatusNotFound, level: "INFO"},
		{name: "server error", status: http.StatusInternalServerError, level: "ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&output, nil))
			router := chi.NewRouter()
			router.Use(middleware.Logging(logger))
			router.Get("/trips/{tripId}", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
			})
			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/trips/123", nil))

			var record map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &record))
			require.Equal(t, tc.level, record["level"])
			require.Equal(t, "/trips/{tripId}", record["route"])
			require.EqualValues(t, tc.status, record["status"])
			require.Equal(t, "request handled", record["msg"])
		})
	}
}

func TestRecoverReturnsInternalProblem(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := middleware.Recover(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("test panic")
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))

	require.Equal(t, http.StatusInternalServerError, response.Code)
	var problem api.Problem
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
	require.Equal(t, "internal_error", problem.Code)
	require.NotContains(t, response.Body.String(), "test panic")
}
