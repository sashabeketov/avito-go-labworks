package model

import (
	"time"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
)

type TripStatusHistory struct {
	ID         int64
	TripID     uuid.UUID
	FromStatus *vobject.TripStatus
	ToStatus   vobject.TripStatus
	Reason     *string
	ChangedAt  time.Time
}
