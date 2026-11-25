package create_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
	"github.com/sashabeketov/avito-go-labworks/internal/usecase/trip/create"
)

func validInput() create.Input {
	return create.Input{
		UserID:     uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c"),
		DriverID:   uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5"),
		StartPoint: vobject.Coordinates{Latitude: 55.75, Longitude: 37.62},
		EndPoint:   vobject.Coordinates{Latitude: 55.76, Longitude: 37.63},
		Price:      1450,
	}
}

func TestCreateSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	trips := NewMocktripRepository(ctrl)
	history := NewMockhistoryRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	clock := NewMockclock(ctrl)
	ids := NewMockidGenerator(ctrl)
	events := NewMocktripEvents(ctrl)
	ctx := context.Background()
	input := validInput()
	id := uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
	now := time.Date(2026, time.November, 6, 12, 0, 0, 0, time.UTC)
	want := model.Trip{
		ID: id, UserID: input.UserID, DriverID: input.DriverID,
		StartPoint: input.StartPoint, EndPoint: input.EndPoint,
		Price: vobject.Money(input.Price), Status: vobject.TripStatusActive, StartedAt: now,
	}
	clock.EXPECT().Now().Return(now)
	ids.EXPECT().New().Return(id)
	tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	trips.EXPECT().Create(ctx, want).Return(nil)
	history.EXPECT().Create(ctx, model.TripStatusHistory{
		TripID: id, ToStatus: vobject.TripStatusActive, ChangedAt: now,
	}).Return(nil)
	trips.EXPECT().GetByID(ctx, id).Return(want, nil)
	events.EXPECT().Created(ctx, want)

	got, err := create.New(trips, history, tx, clock, ids, events).Execute(ctx, input)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestCreateDriverBusyDoesNotEmitEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	trips := NewMocktripRepository(ctrl)
	history := NewMockhistoryRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	clock := NewMockclock(ctrl)
	ids := NewMockidGenerator(ctrl)
	events := NewMocktripEvents(ctrl)
	ctx := context.Background()
	clock.EXPECT().Now().Return(time.Now())
	ids.EXPECT().New().Return(uuid.New())
	tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	trips.EXPECT().Create(ctx, gomock.Any()).Return(trip.ErrDriverHasActiveTrip)

	_, err := create.New(trips, history, tx, clock, ids, events).Execute(ctx, validInput())
	require.ErrorIs(t, err, trip.ErrDriverHasActiveTrip)
}

func TestCreateValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field string
		edit  func(*create.Input)
	}{
		{name: "user", field: "user_id", edit: func(i *create.Input) { i.UserID = uuid.Nil() }},
		{name: "driver", field: "driver_id", edit: func(i *create.Input) { i.DriverID = uuid.Nil() }},
		{name: "start", field: "start_point", edit: func(i *create.Input) { i.StartPoint.Latitude = 91 }},
		{name: "end", field: "end_point", edit: func(i *create.Input) { i.EndPoint.Longitude = 181 }},
		{name: "price", field: "price", edit: func(i *create.Input) { i.Price = -1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			input := validInput()
			tc.edit(&input)
			useCase := create.New(
				NewMocktripRepository(ctrl), NewMockhistoryRepository(ctrl),
				NewMocktxManager(ctrl), NewMockclock(ctrl), NewMockidGenerator(ctrl), NewMocktripEvents(ctrl),
			)
			_, err := useCase.Execute(context.Background(), input)
			require.ErrorIs(t, err, trip.ErrInvalidTrip)
			var validation *trip.ValidationError
			require.True(t, errors.As(err, &validation))
			require.Equal(t, tc.field, validation.Field)
		})
	}
}
