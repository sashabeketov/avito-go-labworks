//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create_test
package create

import (
	"context"
	"time"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type positionRepository interface {
	TouchActiveTrip(ctx context.Context, tripID uuid.UUID, recordedAt, updatedAt time.Time) error
	Create(ctx context.Context, position model.TripPosition) (model.TripPosition, error)
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type clock interface {
	Now() time.Time
}

type positionEvents interface {
	PositionSaved(ctx context.Context, tripID uuid.UUID)
}
