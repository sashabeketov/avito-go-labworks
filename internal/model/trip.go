package model

import (
	"time"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
)

type Trip struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	DriverID uuid.UUID

	StartPoint vobject.Coordinates
	EndPoint   vobject.Coordinates

	Price  vobject.Money
	Status vobject.TripStatus

	StartedAt  time.Time
	FinishedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
