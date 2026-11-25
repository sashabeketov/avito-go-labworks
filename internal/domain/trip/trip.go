package trip

import (
	"fmt"
	"time"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
)

// Trip проверяет допустимость перехода статуса внутри домена.
type Trip struct {
	status     vobject.TripStatus
	finishedAt *time.Time
}

func Restore(status vobject.TripStatus, finishedAt *time.Time) (Trip, error) {
	if _, err := vobject.NewTripStatus(string(status)); err != nil {
		return Trip{}, fmt.Errorf("restore trip status: %w: %w", ErrInvalidTrip, err)
	}
	if (status == vobject.TripStatusActive && finishedAt != nil) ||
		(status == vobject.TripStatusCompleted && finishedAt == nil) {
		return Trip{}, fmt.Errorf("restore trip status and finish time: %w", ErrInvalidTrip)
	}
	return Trip{status: status, finishedAt: finishedAt}, nil
}

func (t *Trip) Finish(at time.Time) error {
	if t.status != vobject.TripStatusActive {
		return ErrTripAlreadyCompleted
	}
	t.status = vobject.TripStatusCompleted
	t.finishedAt = &at
	return nil
}

func (t Trip) Status() vobject.TripStatus {
	return t.status
}

func (t Trip) FinishedAt() *time.Time {
	return t.finishedAt
}
