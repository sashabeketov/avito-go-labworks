package create_test

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
	"github.com/sashabeketov/avito-go-labworks/internal/usecase/position/create"
)

func TestCreatePositionUpdatesTripAndInsertsInTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	positions := NewMockpositionRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	clock := NewMockclock(ctrl)
	events := NewMockpositionEvents(ctrl)
	ctx := context.Background()
	now := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	input := create.Input{TripID: uuid.New(), Coordinates: vobject.Coordinates{Latitude: 59.93, Longitude: 30.33}, RecordedAt: now.Add(-time.Minute)}
	want := model.TripPosition{ID: 42, TripID: input.TripID, Coordinates: input.Coordinates, RecordedAt: input.RecordedAt}
	clock.EXPECT().Now().Return(now)
	committed := false
	tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		err := fn(ctx)
		committed = err == nil
		return err
	})
	gomock.InOrder(
		positions.EXPECT().TouchActiveTrip(ctx, input.TripID, input.RecordedAt, now).Return(nil),
		positions.EXPECT().Create(ctx, model.TripPosition{TripID: input.TripID, Coordinates: input.Coordinates, RecordedAt: input.RecordedAt}).Return(want, nil),
	)

	events.EXPECT().PositionSaved(ctx, input.TripID).Do(func(context.Context, uuid.UUID) {
		require.True(t, committed)
	})
	got, err := create.New(positions, tx, clock, events).Execute(ctx, input)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestCreatePositionRejectsInvalidInput(t *testing.T) {
	now := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		edit func(*create.Input)
	}{
		{"empty trip ID", func(i *create.Input) { i.TripID = uuid.Nil() }},
		{"latitude", func(i *create.Input) { i.Coordinates.Latitude = 91 }},
		{"longitude", func(i *create.Input) { i.Coordinates.Longitude = -181 }},
		{"missing time", func(i *create.Input) { i.RecordedAt = time.Time{} }},
		{"future time", func(i *create.Input) { i.RecordedAt = now.Add(time.Minute + time.Nanosecond) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			clock := NewMockclock(ctrl)
			input := create.Input{TripID: uuid.New(), Coordinates: vobject.Coordinates{Latitude: 59.93, Longitude: 30.33}, RecordedAt: now}
			tc.edit(&input)
			if tc.name == "missing time" || tc.name == "future time" {
				clock.EXPECT().Now().Return(now)
			}
			_, err := create.New(NewMockpositionRepository(ctrl), NewMocktxManager(ctrl), clock, NewMockpositionEvents(ctrl)).Execute(context.Background(), input)
			require.ErrorIs(t, err, trip.ErrInvalidTrip)
		})
	}
}

func TestCreatePositionDoesNotInsertIntoCompletedTrip(t *testing.T) {
	ctrl := gomock.NewController(t)
	positions := NewMockpositionRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	clock := NewMockclock(ctrl)
	ctx := context.Background()
	now := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	input := create.Input{TripID: uuid.New(), Coordinates: vobject.Coordinates{Latitude: 1, Longitude: 2}, RecordedAt: now}
	clock.EXPECT().Now().Return(now)
	tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	positions.EXPECT().TouchActiveTrip(ctx, input.TripID, now, now).Return(trip.ErrTripAlreadyCompleted)
	_, err := create.New(positions, tx, clock, NewMockpositionEvents(ctrl)).Execute(ctx, input)
	require.ErrorIs(t, err, trip.ErrTripAlreadyCompleted)
}
