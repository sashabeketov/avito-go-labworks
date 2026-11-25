//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=positions
package positions

import (
	"context"
	"time"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type staleTripSource interface {
	ListStale(ctx context.Context, staleAfter time.Duration) ([]model.StaleTrip, error)
}

type pushGateway interface {
	RequestPosition(ctx context.Context, trip model.StaleTrip) error
}

type ticker interface {
	Ticks() <-chan time.Time
	Stop()
}
