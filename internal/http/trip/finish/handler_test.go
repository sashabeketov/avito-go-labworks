package finish_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
	"github.com/sashabeketov/avito-go-labworks/internal/http/trip/finish"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

func TestFinishHandlerMapsCompletedTrip(t *testing.T) {
	ctrl := gomock.NewController(t)
	useCase := NewMockuseCase(ctrl)
	id := uuid.New()
	useCase.EXPECT().Execute(gomock.Any(), id).Return(model.Trip{}, trip.ErrTripAlreadyCompleted)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id.String()+"/finish", nil)
	w := httptest.NewRecorder()
	finish.New(useCase, logger).Handle(w, r, api.TripId(id))

	require.Equal(t, http.StatusConflict, w.Code)
	var problem api.Problem
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &problem))
	require.Equal(t, "trip_completed", problem.Code)
}
