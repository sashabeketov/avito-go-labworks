package model

import "uuid"

type StaleTrip struct {
	TripID   uuid.UUID
	DriverID uuid.UUID
}
