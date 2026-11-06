//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create_test
package create

import (
	"context"
	"time"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type tripRepository interface {
	Create(ctx context.Context, trip model.Trip) error
	GetByID(ctx context.Context, id uuid.UUID) (model.Trip, error)
}

type historyRepository interface {
	Create(ctx context.Context, history model.TripStatusHistory) error
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type clock interface {
	Now() time.Time
}

type idGenerator interface {
	New() uuid.UUID
}

type tripEvents interface {
	Created(ctx context.Context, trip model.Trip)
}
