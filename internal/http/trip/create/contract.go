//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create_test
package create

import (
	"context"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
	createuc "github.com/sashabeketov/avito-go-labworks/internal/usecase/trip/create"
)

type useCase interface {
	Execute(ctx context.Context, input createuc.Input) (model.Trip, error)
}
