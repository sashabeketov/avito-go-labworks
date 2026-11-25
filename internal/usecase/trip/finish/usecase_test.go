package finish_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
	"github.com/sashabeketov/avito-go-labworks/internal/usecase/trip/finish"
)

func TestFinishSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	trips := NewMocktripRepository(ctrl)
	history := NewMockhistoryRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	clock := NewMockclock(ctrl)
	events := NewMocktripEvents(ctrl)
	ctx := context.Background()
	id := uuid.New()
	at := time.Date(2026, time.November, 6, 12, 0, 0, 0, time.UTC)
	current := model.Trip{ID: id, Status: vobject.TripStatusActive}
	finished := model.Trip{ID: id, Status: vobject.TripStatusCompleted, FinishedAt: &at}
	tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	trips.EXPECT().GetByID(ctx, id).Return(current, nil)
	clock.EXPECT().Now().Return(at)
	trips.EXPECT().Finish(ctx, id, at).Return(finished, nil)
	from := vobject.TripStatusActive
	history.EXPECT().Create(ctx, model.TripStatusHistory{
		TripID: id, FromStatus: &from, ToStatus: vobject.TripStatusCompleted, ChangedAt: at,
	}).Return(nil)
	events.EXPECT().Completed(ctx, finished)

	got, err := finish.New(trips, history, tx, clock, events).Execute(ctx, id)
	require.NoError(t, err)
	require.Equal(t, finished, got)
}

func TestFinishCompletedDoesNotWriteOrEmitEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	trips := NewMocktripRepository(ctrl)
	history := NewMockhistoryRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	clock := NewMockclock(ctrl)
	events := NewMocktripEvents(ctrl)
	ctx := context.Background()
	id := uuid.New()
	at := time.Now()
	tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	trips.EXPECT().GetByID(ctx, id).Return(model.Trip{
		ID: id, Status: vobject.TripStatusCompleted, FinishedAt: &at,
	}, nil)
	clock.EXPECT().Now().Return(at)

	_, err := finish.New(trips, history, tx, clock, events).Execute(ctx, id)
	require.ErrorIs(t, err, trip.ErrTripAlreadyCompleted)
}

func TestFinishMissingTrip(t *testing.T) {
	ctrl := gomock.NewController(t)
	trips := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	ctx := context.Background()
	id := uuid.New()
	tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	trips.EXPECT().GetByID(ctx, id).Return(model.Trip{}, trip.ErrTripNotFound)
	_, err := finish.New(
		trips, NewMockhistoryRepository(ctrl), tx, NewMockclock(ctrl), NewMocktripEvents(ctrl),
	).Execute(ctx, id)
	require.ErrorIs(t, err, trip.ErrTripNotFound)
}
