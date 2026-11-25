//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=list_test
package list

import (
	"context"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type tripRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (model.Trip, error)
}

type positionRepository interface {
	ListByTrip(ctx context.Context, tripID uuid.UUID) ([]model.TripPosition, error)
}
