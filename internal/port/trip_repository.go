package port

import (
	"context"
	"time"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/domain"
)

type TripRepository interface {
	Create(ctx context.Context, trip domain.Trip) error

	GetByID(ctx context.Context, id uuid.UUID) (domain.Trip, error)

	Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (domain.Trip, error)
}
