//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=finish_test
package finish

import (
	"context"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type useCase interface {
	Execute(ctx context.Context, id uuid.UUID) (model.Trip, error)
}
