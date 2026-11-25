//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=finish_test
package finish

import (
	"context"
	"time"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type tripRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (model.Trip, error)
	Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (model.Trip, error)
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

type tripEvents interface {
	Completed(ctx context.Context, trip model.Trip)
}
