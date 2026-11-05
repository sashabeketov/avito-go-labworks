package vobject

import "errors"

type TripStatus string

const (
	TripStatusActive    TripStatus = "active"
	TripStatusCompleted TripStatus = "completed"
)

func NewTripStatus(value string) (TripStatus, error) {
	switch TripStatus(value) {
	case TripStatusActive, TripStatusCompleted:
		return TripStatus(value), nil
	default:
		return "", errors.New("invalid trip status")
	}
}
