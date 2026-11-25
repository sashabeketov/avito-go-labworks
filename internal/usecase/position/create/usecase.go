package create

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type Input struct {
	TripID      uuid.UUID
	Coordinates vobject.Coordinates
	RecordedAt  time.Time
}

type UseCase struct {
	positions positionRepository
	tx        txManager
	clock     clock
	events    positionEvents
}

func New(positions positionRepository, tx txManager, clock clock, events positionEvents) *UseCase {
	return &UseCase{positions: positions, tx: tx, clock: clock, events: events}
}

func (u *UseCase) Execute(ctx context.Context, input Input) (model.TripPosition, error) {
	if input.TripID == uuid.Nil() {
		return model.TripPosition{}, &trip.ValidationError{Field: "trip_id", Reason: "must not be empty"}
	}
	coordinates, err := vobject.NewCoordinates(input.Coordinates.Latitude, input.Coordinates.Longitude)
	if err != nil {
		return model.TripPosition{}, &trip.ValidationError{Field: "coordinates", Reason: err.Error(), Cause: err}
	}
	now := u.clock.Now().UTC()
	if input.RecordedAt.IsZero() || input.RecordedAt.After(now.Add(time.Minute)) {
		return model.TripPosition{}, &trip.ValidationError{Field: "recorded_at", Reason: "must be valid and no more than one minute in the future"}
	}

	position := model.TripPosition{
		TripID: input.TripID, Coordinates: coordinates, RecordedAt: input.RecordedAt.UTC(),
	}
	var saved model.TripPosition
	err = u.tx.Do(ctx, func(txCtx context.Context) error {
		if err := u.positions.TouchActiveTrip(txCtx, input.TripID, position.RecordedAt, now); err != nil {
			return fmt.Errorf("check and update active trip: %w", err)
		}
		var err error
		saved, err = u.positions.Create(txCtx, position)
		if err != nil {
			return fmt.Errorf("save position: %w", err)
		}
		return nil
	})
	if err != nil {
		return model.TripPosition{}, fmt.Errorf("create trip position: %w", err)
	}
	u.events.PositionSaved(ctx, saved.TripID)
	return saved, nil
}
