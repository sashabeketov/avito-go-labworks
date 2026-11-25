package list_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
	"github.com/sashabeketov/avito-go-labworks/internal/http/position/list"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

func TestListPositionsResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	useCase := NewMockuseCase(ctrl)
	id := uuid.New()
	position := model.TripPosition{ID: 42, TripID: id, Coordinates: vobject.Coordinates{Latitude: 59.93, Longitude: 30.33}, RecordedAt: time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)}
	useCase.EXPECT().Execute(gomock.Any(), id).Return([]model.TripPosition{position}, nil)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id.String()+"/positions", nil)
	w := httptest.NewRecorder()
	list.New(useCase, slog.New(slog.NewTextHandler(io.Discard, nil))).Handle(w, r, api.TripId(id))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var response api.TripPositionsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Len(t, response.Positions, 1)
	require.Equal(t, int64(42), response.Positions[0].Id)
}

func TestListPositionsReturnsEmptyArray(t *testing.T) {
	ctrl := gomock.NewController(t)
	useCase := NewMockuseCase(ctrl)
	id := uuid.New()
	useCase.EXPECT().Execute(gomock.Any(), id).Return(nil, nil)
	r := httptest.NewRequest(http.MethodGet, "/positions", nil)
	w := httptest.NewRecorder()
	list.New(useCase, slog.New(slog.NewTextHandler(io.Discard, nil))).Handle(w, r, api.TripId(id))
	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, `{"positions":[]}`, w.Body.String())
}

func TestListPositionsReturnsNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	useCase := NewMockuseCase(ctrl)
	id := uuid.New()
	useCase.EXPECT().Execute(gomock.Any(), id).Return(nil, trip.ErrTripNotFound)
	r := httptest.NewRequest(http.MethodGet, "/positions", nil)
	w := httptest.NewRecorder()
	list.New(useCase, slog.New(slog.NewTextHandler(io.Discard, nil))).Handle(w, r, api.TripId(id))
	require.Equal(t, http.StatusNotFound, w.Code)
	var problem api.Problem
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &problem))
	require.Equal(t, "trip_not_found", problem.Code)
}
