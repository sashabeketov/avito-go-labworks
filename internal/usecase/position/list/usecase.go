package list

import (
	"context"
	"fmt"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type UseCase struct {
	trips     tripRepository
	positions positionRepository
}

func New(trips tripRepository, positions positionRepository) *UseCase {
	return &UseCase{trips: trips, positions: positions}
}

func (u *UseCase) Execute(ctx context.Context, tripID uuid.UUID) ([]model.TripPosition, error) {
	if tripID == uuid.Nil() {
		return nil, &trip.ValidationError{Field: "trip_id", Reason: "must not be empty"}
	}
	if _, err := u.trips.GetByID(ctx, tripID); err != nil {
		return nil, fmt.Errorf("get trip before listing positions: %w", err)
	}
	positions, err := u.positions.ListByTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("list trip positions: %w", err)
	}
	return positions, nil
}
