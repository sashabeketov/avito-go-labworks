package vobject_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
)

func TestCoordinates(t *testing.T) {
	for _, tc := range []struct {
		name string
		lat  float64
		lon  float64
		ok   bool
	}{
		{name: "zero", ok: true},
		{name: "boundaries", lat: 90, lon: -180, ok: true},
		{name: "latitude too high", lat: 90.1},
		{name: "longitude too low", lon: -180.1},
		{name: "nan latitude", lat: math.NaN()},
		{name: "nan longitude", lon: math.NaN()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := vobject.NewCoordinates(tc.lat, tc.lon)
			if tc.ok {
				require.NoError(t, err)
				require.Equal(t, tc.lat, value.Latitude)
				require.Equal(t, tc.lon, value.Longitude)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestMoneyAndStatus(t *testing.T) {
	money, err := vobject.NewMoney(0)
	require.NoError(t, err)
	require.Equal(t, vobject.Money(0), money)
	_, err = vobject.NewMoney(-1)
	require.Error(t, err)

	for _, value := range []string{"active", "completed"} {
		status, err := vobject.NewTripStatus(value)
		require.NoError(t, err)
		require.Equal(t, vobject.TripStatus(value), status)
	}
	_, err = vobject.NewTripStatus("unknown")
	require.Error(t, err)
}
