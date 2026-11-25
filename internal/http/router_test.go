package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	tripdomain "github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
	"github.com/sashabeketov/avito-go-labworks/internal/http/health"
	createhttp "github.com/sashabeketov/avito-go-labworks/internal/http/trip/create"
	finishhttp "github.com/sashabeketov/avito-go-labworks/internal/http/trip/finish"
	gethttp "github.com/sashabeketov/avito-go-labworks/internal/http/trip/get"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
	createuc "github.com/sashabeketov/avito-go-labworks/internal/usecase/trip/create"
)

const (
	validCreateBody = `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":59.9398,"longitude":30.3146},"end_point":{"latitude":59.929,"longitude":30.3626},"price":1450}`
	targetGet       = "get"
	targetFinish    = "finish"
)

type (
	mockCreateUseCase struct{ mock.Mock }
	mockGetUseCase    struct{ mock.Mock }
	mockFinishUseCase struct{ mock.Mock }
)

func (m *mockCreateUseCase) Execute(ctx context.Context, input createuc.Input) (model.Trip, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(model.Trip), args.Error(1)
}

func (m *mockGetUseCase) Execute(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(model.Trip), args.Error(1)
}

func (m *mockFinishUseCase) Execute(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(model.Trip), args.Error(1)
}

func tripHandler(create *mockCreateUseCase, get *mockGetUseCase, finish *mockFinishUseCase) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler, err := NewRouter(
		createhttp.New(create, logger), gethttp.New(get, logger), finishhttp.New(finish, logger),
		health.New(nil), logger, otel.GetTracerProvider(), otel.GetMeterProvider(), propagation.TraceContext{},
	)
	if err != nil {
		panic(err)
	}
	return handler
}

func TestCreateTripResponse(t *testing.T) {
	id := uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
	userID := uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c")
	driverID := uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5")
	startedAt := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)

	expectedInput := createuc.Input{
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: vobject.Coordinates{Latitude: 59.9398, Longitude: 30.3146},
		EndPoint:   vobject.Coordinates{Latitude: 59.929, Longitude: 30.3626},
		Price:      1450,
	}
	expectedTrip := model.Trip{
		ID:         id,
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: expectedInput.StartPoint,
		EndPoint:   expectedInput.EndPoint,
		Price:      1450,
		Status:     vobject.TripStatusActive,
		StartedAt:  startedAt,
	}
	trips := new(mockCreateUseCase)
	trips.On("Execute", mock.Anything, expectedInput).Return(expectedTrip, nil).Once()

	r := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(validCreateBody))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	tripHandler(trips, nil, nil).ServeHTTP(w, r)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Equal(t, "/api/v1/trips/"+id.String(), w.Header().Get("Location"))
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response api.Trip
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, id.String(), response.Id.String())
	require.Equal(t, api.Active, response.Status)
	require.Equal(t, int64(1450), response.Price)
	trips.AssertExpectations(t)
}

func TestCreateTripRejectsInvalidBody(t *testing.T) {
	base := validCreateBody
	cases := []struct {
		name string
		body string
	}{
		{name: "missing price", body: strings.Replace(base, `,"price":1450`, "", 1)},
		{name: "missing latitude", body: strings.Replace(base, `"latitude":59.9398,`, "", 1)},
		{name: "unknown field", body: strings.Replace(base, `"price":1450`, `"price":1450,"extra":true`, 1)},
		{name: "null price", body: strings.Replace(base, `"price":1450`, `"price":null`, 1)},
		{name: "extra JSON value", body: base + ` {}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trips := new(mockCreateUseCase)
			r := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			tripHandler(trips, nil, nil).ServeHTTP(w, r)

			checkProblem(t, w, http.StatusBadRequest, "invalid_request")
			trips.AssertNotCalled(t, "Execute", mock.Anything, mock.Anything)
		})
	}
}

func TestCreateTripErrorResponses(t *testing.T) {
	body := validCreateBody
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "driver busy", err: fmt.Errorf("create trip: %w", tripdomain.ErrDriverHasActiveTrip), status: 409, code: "driver_busy"},
		{name: "internal error", err: errors.New("private database detail"), status: 500, code: "internal_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trips := new(mockCreateUseCase)
			trips.On("Execute", mock.Anything, mock.Anything).Return(model.Trip{}, tc.err).Once()

			r := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			tripHandler(trips, nil, nil).ServeHTTP(w, r)

			checkProblem(t, w, tc.status, tc.code)
			require.NotContains(t, w.Body.String(), "private database detail")
			trips.AssertExpectations(t)
		})
	}
}

func TestTripErrors(t *testing.T) {
	id := uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
	for _, tc := range []struct {
		name, method, path, target string
		serviceErr                 error
		status                     int
		code                       string
	}{
		{name: "not found", method: http.MethodGet, path: "/api/v1/trips/" + id.String(), target: targetGet, serviceErr: fmt.Errorf("get trip: %w", tripdomain.ErrTripNotFound), status: 404, code: "trip_not_found"},
		{name: "completed", method: http.MethodPost, path: "/api/v1/trips/" + id.String() + "/finish", target: targetFinish, serviceErr: tripdomain.ErrTripAlreadyCompleted, status: 409, code: "trip_completed"},
		{name: "invalid UUID", method: http.MethodGet, path: "/api/v1/trips/not-a-uuid", status: 400, code: "invalid_request"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			getTrips := new(mockGetUseCase)
			finishTrips := new(mockFinishUseCase)
			switch tc.target {
			case targetGet:
				getTrips.On("Execute", mock.Anything, id).Return(model.Trip{}, tc.serviceErr).Once()
			case targetFinish:
				finishTrips.On("Execute", mock.Anything, id).Return(model.Trip{}, tc.serviceErr).Once()
			}
			r := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			tripHandler(nil, getTrips, finishTrips).ServeHTTP(w, r)
			checkProblem(t, w, tc.status, tc.code)
			getTrips.AssertExpectations(t)
			finishTrips.AssertExpectations(t)
			if tc.target == "" {
				getTrips.AssertNotCalled(t, "Execute", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestGetAndFinishTripResponses(t *testing.T) {
	id := uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
	finishedAt := time.Date(2026, time.August, 24, 12, 15, 0, 0, time.UTC)
	trip := model.Trip{ID: id, Status: vobject.TripStatusCompleted, FinishedAt: &finishedAt}
	for _, tc := range []struct{ name, method, path, target string }{
		{name: targetGet, method: http.MethodGet, path: "/api/v1/trips/" + id.String(), target: targetGet},
		{name: targetFinish, method: http.MethodPost, path: "/api/v1/trips/" + id.String() + "/finish", target: targetFinish},
	} {
		t.Run(tc.name, func(t *testing.T) {
			getTrips := new(mockGetUseCase)
			finishTrips := new(mockFinishUseCase)
			if tc.target == targetGet {
				getTrips.On("Execute", mock.Anything, id).Return(trip, nil).Once()
			} else {
				finishTrips.On("Execute", mock.Anything, id).Return(trip, nil).Once()
			}
			r := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			tripHandler(nil, getTrips, finishTrips).ServeHTTP(w, r)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.Equal(t, "application/json", w.Header().Get("Content-Type"))
			var response api.Trip
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			require.Equal(t, id.String(), response.Id.String())
			require.Equal(t, api.Completed, response.Status)
			require.NotNil(t, response.FinishedAt)
			require.True(t, response.FinishedAt.Equal(finishedAt))
			getTrips.AssertExpectations(t)
			finishTrips.AssertExpectations(t)
		})
	}
}

func checkProblem(t *testing.T, w *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, w.Code, w.Body.String())
	require.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

	var problem api.Problem
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &problem))
	require.Equal(t, code, problem.Code)
	require.EqualValues(t, status, problem.Status)
}
