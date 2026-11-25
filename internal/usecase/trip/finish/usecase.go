package finish

import (
	"context"
	"fmt"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type UseCase struct {
	trips   tripRepository
	history historyRepository
	tx      txManager
	clock   clock
	events  tripEvents
}

func New(
	trips tripRepository,
	history historyRepository,
	tx txManager,
	clock clock,
	events tripEvents,
) *UseCase {
	return &UseCase{
		trips:   trips,
		history: history,
		tx:      tx,
		clock:   clock,
		events:  events,
	}
}

func (u *UseCase) Execute(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	var finished model.Trip
	err := u.tx.Do(ctx, func(txCtx context.Context) error {
		current, err := u.trips.GetByID(txCtx, id)
		if err != nil {
			return fmt.Errorf("read trip before finish: %w", err)
		}

		state, err := trip.Restore(current.Status, current.FinishedAt)
		if err != nil {
			return fmt.Errorf("restore trip before finish: %w", err)
		}

		finishedAt := u.clock.Now().UTC()
		if err := state.Finish(finishedAt); err != nil {
			return fmt.Errorf("finish trip state: %w", err)
		}

		finished, err = u.trips.Finish(txCtx, id, finishedAt)
		if err != nil {
			return fmt.Errorf("update trip: %w", err)
		}

		fromStatus := vobject.TripStatusActive
		if err := u.history.Create(txCtx, model.TripStatusHistory{
			TripID:     id,
			FromStatus: &fromStatus,
			ToStatus:   state.Status(),
			ChangedAt:  finishedAt,
		}); err != nil {
			return fmt.Errorf("insert trip status history: %w", err)
		}

		return nil
	})
	if err != nil {
		return model.Trip{}, fmt.Errorf("finish trip: %w", err)
	}

	u.events.Completed(ctx, finished)
	return finished, nil
}
