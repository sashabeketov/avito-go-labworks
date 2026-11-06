package create

import (
	"context"
	"fmt"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type Input struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint vobject.Coordinates
	EndPoint   vobject.Coordinates
	Price      int64
}

type UseCase struct {
	trips   tripRepository
	history historyRepository
	tx      txManager
	clock   clock
	ids     idGenerator
	events  tripEvents
}

func New(
	trips tripRepository,
	history historyRepository,
	tx txManager,
	clock clock,
	ids idGenerator,
	events tripEvents,
) *UseCase {
	return &UseCase{
		trips:   trips,
		history: history,
		tx:      tx,
		clock:   clock,
		ids:     ids,
		events:  events,
	}
}

func (u *UseCase) Execute(ctx context.Context, input Input) (model.Trip, error) {
	if input.UserID == uuid.Nil() {
		return model.Trip{}, &trip.ValidationError{Field: "user_id", Reason: "must not be empty"}
	}
	if input.DriverID == uuid.Nil() {
		return model.Trip{}, &trip.ValidationError{Field: "driver_id", Reason: "must not be empty"}
	}

	startPoint, err := vobject.NewCoordinates(input.StartPoint.Latitude, input.StartPoint.Longitude)
	if err != nil {
		return model.Trip{}, &trip.ValidationError{Field: "start_point", Reason: err.Error(), Cause: err}
	}

	endPoint, err := vobject.NewCoordinates(input.EndPoint.Latitude, input.EndPoint.Longitude)
	if err != nil {
		return model.Trip{}, &trip.ValidationError{Field: "end_point", Reason: err.Error(), Cause: err}
	}

	price, err := vobject.NewMoney(input.Price)
	if err != nil {
		return model.Trip{}, &trip.ValidationError{Field: "price", Reason: err.Error(), Cause: err}
	}

	now := u.clock.Now().UTC()
	newTrip := model.Trip{
		ID:         u.ids.New(),
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		StartPoint: startPoint,
		EndPoint:   endPoint,
		Price:      price,
		Status:     vobject.TripStatusActive,
		StartedAt:  now,
	}

	var created model.Trip
	err = u.tx.Do(ctx, func(txCtx context.Context) error {
		if err := u.trips.Create(txCtx, newTrip); err != nil {
			return fmt.Errorf("insert trip: %w", err)
		}
		if err := u.history.Create(txCtx, model.TripStatusHistory{
			TripID:    newTrip.ID,
			ToStatus:  vobject.TripStatusActive,
			ChangedAt: now,
		}); err != nil {
			return fmt.Errorf("insert trip status history: %w", err)
		}

		var err error
		created, err = u.trips.GetByID(txCtx, newTrip.ID)
		if err != nil {
			return fmt.Errorf("read created trip: %w", err)
		}

		return nil
	})
	if err != nil {
		return model.Trip{}, fmt.Errorf("create trip: %w", err)
	}

	u.events.Created(ctx, created)
	return created, nil
}
