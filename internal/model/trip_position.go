package model

import (
	"time"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
)

type TripPosition struct {
	ID          int64
	TripID      uuid.UUID
	Coordinates vobject.Coordinates
	RecordedAt  time.Time
}
