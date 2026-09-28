package vobject

import (
	"errors"
	"math"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

func NewCoordinates(
	lat float64,
	lon float64,
) (Coordinates, error) {
	if math.IsNaN(lat) || lat < -90 || lat > 90 {
		return Coordinates{}, errors.New("invalid latitude")
	}

	if math.IsNaN(lon) || lon < -180 || lon > 180 {
		return Coordinates{}, errors.New("invalid longitude")
	}

	return Coordinates{
		Latitude:  lat,
		Longitude: lon,
	}, nil
}
