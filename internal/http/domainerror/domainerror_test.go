package domainerror_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	"github.com/sashabeketov/avito-go-labworks/internal/http/domainerror"
)

func TestProblemMapsDomainErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "validation", err: &trip.ValidationError{Field: "price", Reason: "negative"}, status: http.StatusBadRequest, code: "invalid_request"},
		{name: "not found", err: trip.ErrTripNotFound, status: http.StatusNotFound, code: "trip_not_found"},
		{name: "driver busy", err: trip.ErrDriverHasActiveTrip, status: http.StatusConflict, code: "driver_busy"},
		{name: "completed", err: trip.ErrTripAlreadyCompleted, status: http.StatusConflict, code: "trip_completed"},
		{name: "internal", err: errors.New("secret database detail"), status: http.StatusInternalServerError, code: "internal_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := domainerror.Problem(fmt.Errorf("usecase: %w", tc.err))
			require.Equal(t, tc.status, status)
			require.Equal(t, tc.code, body.Code)
			require.EqualValues(t, status, body.Status)
			require.NotContains(t, body.Title, "secret database detail")
			require.NotContains(t, *body.Detail, "secret database detail")
		})
	}
}
