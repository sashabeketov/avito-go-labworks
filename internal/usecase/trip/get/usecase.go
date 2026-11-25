package get

import (
	"context"
	"fmt"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type UseCase struct{ trips tripRepository }

func New(trips tripRepository) *UseCase { return &UseCase{trips: trips} }

func (u *UseCase) Execute(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	result, err := u.trips.GetByID(ctx, id)
	if err != nil {
		return model.Trip{}, fmt.Errorf("get trip: %w", err)
	}
	return result, nil
}
