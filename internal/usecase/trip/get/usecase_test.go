package get_test

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/trip"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
	"github.com/sashabeketov/avito-go-labworks/internal/usecase/trip/get"
)

func TestGetTrip(t *testing.T) {
	id := uuid.New()
	for _, tc := range []struct {
		name string
		trip model.Trip
		err  error
	}{
		{name: "found", trip: model.Trip{ID: id}},
		{name: "missing", err: trip.ErrTripNotFound},
		{name: "database failure", err: errors.New("database unavailable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			ctx := context.Background()
			repo.EXPECT().GetByID(ctx, id).Return(tc.trip, tc.err)
			got, err := get.New(repo).Execute(ctx, id)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				require.Equal(t, model.Trip{}, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.trip, got)
		})
	}
}
