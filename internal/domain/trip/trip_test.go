package trip_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
)

func TestFinish(t *testing.T) {
	at := time.Date(2026, time.November, 6, 12, 0, 0, 0, time.UTC)
	state, err := trip.Restore(vobject.TripStatusActive, nil)
	require.NoError(t, err)

	require.NoError(t, state.Finish(at))
	require.Equal(t, vobject.TripStatusCompleted, state.Status())
	require.Equal(t, &at, state.FinishedAt())
	require.ErrorIs(t, state.Finish(at.Add(time.Hour)), trip.ErrTripAlreadyCompleted)
	require.Equal(t, &at, state.FinishedAt(), "a repeated finish must not move the timestamp")
}

func TestRestoreRejectsInconsistentState(t *testing.T) {
	at := time.Now()
	for _, tc := range []struct {
		name       string
		status     vobject.TripStatus
		finishedAt *time.Time
	}{
		{name: "unknown status", status: "unknown"},
		{name: "active with finish time", status: vobject.TripStatusActive, finishedAt: &at},
		{name: "completed without finish time", status: vobject.TripStatusCompleted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := trip.Restore(tc.status, tc.finishedAt)
			require.ErrorIs(t, err, trip.ErrInvalidTrip)
		})
	}
}

func TestValidationErrorKeepsTypeAndCause(t *testing.T) {
	cause := errors.New("invalid coordinate")
	err := &trip.ValidationError{Field: "start_point", Reason: "invalid", Cause: cause}
	require.ErrorIs(t, err, trip.ErrInvalidTrip)
	require.ErrorIs(t, err, cause)
	require.Contains(t, err.Error(), "start_point")
}
