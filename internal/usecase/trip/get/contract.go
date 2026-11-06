//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=get_test
package get

import (
	"context"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type tripRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (model.Trip, error)
}
