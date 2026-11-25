package list_test

import (
	"context"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
	"github.com/sashabeketov/avito-go-labworks/internal/usecase/position/list"
)

func TestListPositionsChecksTripBeforeReadingRoute(t *testing.T) {
	ctrl := gomock.NewController(t)
	trips := NewMocktripRepository(ctrl)
	positions := NewMockpositionRepository(ctrl)
	ctx := context.Background()
	id := uuid.New()
	trips.EXPECT().GetByID(ctx, id).Return(model.Trip{ID: id}, nil)
	positions.EXPECT().ListByTrip(ctx, id).Return([]model.TripPosition{}, nil)
	got, err := list.New(trips, positions).Execute(ctx, id)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestListPositionsReturnsNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	trips := NewMocktripRepository(ctrl)
	positions := NewMockpositionRepository(ctrl)
	ctx := context.Background()
	id := uuid.New()
	trips.EXPECT().GetByID(ctx, id).Return(model.Trip{}, trip.ErrTripNotFound)
	_, err := list.New(trips, positions).Execute(ctx, id)
	require.ErrorIs(t, err, trip.ErrTripNotFound)
}
