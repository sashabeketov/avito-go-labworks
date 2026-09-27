package domain

import "errors"

var (
	ErrTripNotFound = errors.New("trip not found")

	ErrDriverHasActiveTrip = errors.New(
		"driver already has active trip",
	)

	ErrTripAlreadyCompleted = errors.New(
		"trip already completed",
	)
)
