package domain

import "errors"

var (
	ErrInvalidTrip = errors.New("invalid trip")

	ErrTripNotFound = errors.New("trip not found")

	ErrDriverHasActiveTrip = errors.New(
		"driver already has active trip",
	)

	ErrTripAlreadyCompleted = errors.New(
		"trip already completed",
	)
)
