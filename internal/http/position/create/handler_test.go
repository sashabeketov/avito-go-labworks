package create_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
	"github.com/sashabeketov/avito-go-labworks/internal/http/position/create"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
	createuc "github.com/sashabeketov/avito-go-labworks/internal/usecase/position/create"
)

func TestCreatePositionResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	useCase := NewMockuseCase(ctrl)
	id := uuid.New()
	recordedAt := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	coordinates := vobject.Coordinates{Latitude: 59.93, Longitude: 30.33}
	useCase.EXPECT().Execute(gomock.Any(), createuc.Input{TripID: id, Coordinates: coordinates, RecordedAt: recordedAt}).Return(model.TripPosition{
		ID: 42, TripID: id, Coordinates: coordinates, RecordedAt: recordedAt,
	}, nil)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id.String()+"/positions", strings.NewReader(`{"coordinates":{"latitude":59.93,"longitude":30.33},"recorded_at":"2026-11-23T12:00:00Z"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	create.New(useCase, slog.New(slog.NewTextHandler(io.Discard, nil))).Handle(w, r, api.TripId(id))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var response api.TripPosition
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, int64(42), response.Id)
	require.Equal(t, id, uuid.UUID(response.TripId))
	require.Equal(t, coordinates.Latitude, response.Coordinates.Latitude)
}

func TestCreatePositionRejectsInvalidJSON(t *testing.T) {
	for _, body := range []string{
		`{"coordinates":{"latitude":1,"longitude":2}}`,
		`{"recorded_at":"2026-11-23T12:00:00Z"}`,
		`{"coordinates":null,"recorded_at":"2026-11-23T12:00:00Z"}`,
		`{"coordinates":{"latitude":1},"recorded_at":"2026-11-23T12:00:00Z"}`,
		`{"coordinates":{"latitude":1,"longitude":2},"recorded_at":null}`,
		`{"coordinates":{"latitude":1,"longitude":2},"recorded_at":"bad"}`,
		`{"coordinates":{"latitude":1,"longitude":2},"recorded_at":"2026-11-23T12:00:00Z","extra":true}`,
	} {
		t.Run(body, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			r := httptest.NewRequest(http.MethodPost, "/positions", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			create.New(NewMockuseCase(ctrl), slog.New(slog.NewTextHandler(io.Discard, nil))).Handle(w, r, api.TripId(uuid.New()))
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			var problem api.Problem
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &problem))
			require.Equal(t, "invalid_request", problem.Code)
		})
	}
}

func TestCreatePositionMapsCompletedTrip(t *testing.T) {
	ctrl := gomock.NewController(t)
	useCase := NewMockuseCase(ctrl)
	useCase.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(model.TripPosition{}, trip.ErrTripAlreadyCompleted)
	r := httptest.NewRequest(http.MethodPost, "/positions", strings.NewReader(`{"coordinates":{"latitude":1,"longitude":2},"recorded_at":"2026-11-23T12:00:00Z"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	create.New(useCase, slog.New(slog.NewTextHandler(io.Discard, nil))).Handle(w, r, api.TripId(uuid.New()))
	require.Equal(t, http.StatusConflict, w.Code)
	var problem api.Problem
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &problem))
	require.Equal(t, "trip_completed", problem.Code)
}
