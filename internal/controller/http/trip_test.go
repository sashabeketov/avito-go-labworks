package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/domain"
	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
	"github.com/sashabeketov/avito-go-labworks/internal/service"
)

type stubTripService struct {
	create func(context.Context, service.CreateTripInput) (domain.Trip, error)
	get    func(context.Context, uuid.UUID) (domain.Trip, error)
	finish func(context.Context, uuid.UUID) (domain.Trip, error)
}

func (s stubTripService) Create(ctx context.Context, input service.CreateTripInput) (domain.Trip, error) {
	return s.create(ctx, input)
}

func (s stubTripService) GetByID(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	return s.get(ctx, id)
}

func (s stubTripService) Finish(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	return s.finish(ctx, id)
}

func tripHandler(trips TripService) http.Handler {
	return api.HandlerWithOptions(NewHandler(nil, trips), api.ChiServerOptions{
		ErrorHandlerFunc: WriteParameterError,
	})
}

func TestCreateTripResponse(t *testing.T) {
	id := uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
	userID := uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c")
	driverID := uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5")
	startedAt := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)

	trips := stubTripService{
		create: func(_ context.Context, input service.CreateTripInput) (domain.Trip, error) {
			if input.UserID != userID || input.DriverID != driverID || input.Price != 1450 {
				t.Fatalf("wrong input: %+v", input)
			}

			return domain.Trip{
				ID:         id,
				UserID:     userID,
				DriverID:   driverID,
				StartPoint: vobject.Coordinates{Latitude: 59.9398, Longitude: 30.3146},
				EndPoint:   vobject.Coordinates{Latitude: 59.929, Longitude: 30.3626},
				Price:      1450,
				Status:     vobject.TripStatusActive,
				StartedAt:  startedAt,
			}, nil
		},
	}

	body := `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":59.9398,"longitude":30.3146},"end_point":{"latitude":59.929,"longitude":30.3626},"price":1450}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	tripHandler(trips).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Location"); got != "/api/v1/trips/"+id.String() {
		t.Fatalf("Location = %q", got)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}

	var response api.Trip
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Id.String() != id.String() || response.Status != api.Active || response.Price != 1450 {
		t.Fatalf("unexpected trip response: %+v", response)
	}
}

func TestCreateTripRejectsInvalidBody(t *testing.T) {
	base := `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":59.9398,"longitude":30.3146},"end_point":{"latitude":59.929,"longitude":30.3626},"price":1450}`
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
			r := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			tripHandler(stubTripService{}).ServeHTTP(w, r)

			checkProblem(t, w, http.StatusBadRequest, "invalid_request")
		})
	}
}

func TestCreateTripErrorResponses(t *testing.T) {
	body := `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":59.9398,"longitude":30.3146},"end_point":{"latitude":59.929,"longitude":30.3626},"price":1450}`
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "driver busy", err: fmt.Errorf("create trip: %w", domain.ErrDriverHasActiveTrip), status: 409, code: "driver_busy"},
		{name: "internal error", err: errors.New("private database detail"), status: 500, code: "internal_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trips := stubTripService{
				create: func(context.Context, service.CreateTripInput) (domain.Trip, error) {
					return domain.Trip{}, tc.err
				},
			}
			r := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			tripHandler(trips).ServeHTTP(w, r)

			checkProblem(t, w, tc.status, tc.code)
			if strings.Contains(w.Body.String(), "private database detail") {
				t.Fatal("internal error leaked into the response")
			}
		})
	}
}

func TestTripErrors(t *testing.T) {
	id := uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
	trips := stubTripService{
		get: func(_ context.Context, got uuid.UUID) (domain.Trip, error) {
			if got != id {
				t.Fatalf("wrong trip ID: %s", got)
			}
			return domain.Trip{}, fmt.Errorf("get trip: %w", domain.ErrTripNotFound)
		},
		finish: func(_ context.Context, got uuid.UUID) (domain.Trip, error) {
			if got != id {
				t.Fatalf("wrong trip ID: %s", got)
			}
			return domain.Trip{}, domain.ErrTripAlreadyCompleted
		},
	}

	for _, tc := range []struct {
		name   string
		method string
		path   string
		status int
		code   string
	}{
		{name: "not found", method: http.MethodGet, path: "/api/v1/trips/" + id.String(), status: 404, code: "trip_not_found"},
		{name: "completed", method: http.MethodPost, path: "/api/v1/trips/" + id.String() + "/finish", status: 409, code: "trip_completed"},
		{name: "invalid UUID", method: http.MethodGet, path: "/api/v1/trips/not-a-uuid", status: 400, code: "invalid_request"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			tripHandler(trips).ServeHTTP(w, r)
			checkProblem(t, w, tc.status, tc.code)
		})
	}
}

func TestGetAndFinishTripResponses(t *testing.T) {
	id := uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
	finishedAt := time.Date(2026, time.August, 24, 12, 15, 0, 0, time.UTC)
	trip := domain.Trip{
		ID:         id,
		Status:     vobject.TripStatusCompleted,
		FinishedAt: &finishedAt,
	}
	trips := stubTripService{
		get: func(_ context.Context, got uuid.UUID) (domain.Trip, error) {
			if got != id {
				t.Fatalf("wrong trip ID: %s", got)
			}
			return trip, nil
		},
		finish: func(_ context.Context, got uuid.UUID) (domain.Trip, error) {
			if got != id {
				t.Fatalf("wrong trip ID: %s", got)
			}
			return trip, nil
		},
	}

	for _, tc := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "get", method: http.MethodGet, path: "/api/v1/trips/" + id.String()},
		{name: "finish", method: http.MethodPost, path: "/api/v1/trips/" + id.String() + "/finish"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			tripHandler(trips).ServeHTTP(w, r)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}
			if got := w.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q", got)
			}
			var response api.Trip
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Id.String() != id.String() || response.Status != api.Completed || response.FinishedAt == nil || !response.FinishedAt.Equal(finishedAt) {
				t.Fatalf("unexpected trip response: %+v", response)
			}
		})
	}
}

func checkProblem(t *testing.T, w *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Content-Type = %q", got)
	}
	var problem api.Problem
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if problem.Code != code || problem.Status != int32(status) {
		t.Fatalf("problem = %+v", problem)
	}
}
