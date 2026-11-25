package create_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
	"github.com/sashabeketov/avito-go-labworks/internal/http/trip/create"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

func TestCreateHandlerRejectsMissingPriceBeforeUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	useCase := NewMockuseCase(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	body := `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":0,"longitude":0},"end_point":{"latitude":0,"longitude":0}}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	create.New(useCase, logger).Handle(w, r, api.CreateTripParams{})

	require.Equal(t, http.StatusBadRequest, w.Code)
	var problem api.Problem
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &problem))
	require.Equal(t, "invalid_request", problem.Code)
}

func TestCreateHandlerAcceptsZeroPriceAndCoordinates(t *testing.T) {
	ctrl := gomock.NewController(t)
	useCase := NewMockuseCase(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	id := uuid.New()
	useCase.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(model.Trip{ID: id}, nil)
	body := `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":0,"longitude":0},"end_point":{"latitude":0,"longitude":0},"price":0}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	create.New(useCase, logger).Handle(w, r, api.CreateTripParams{})

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "/api/v1/trips/"+id.String(), w.Header().Get("Location"))
}
