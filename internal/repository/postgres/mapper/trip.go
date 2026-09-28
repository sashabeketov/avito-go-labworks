package mapper

import (
	"fmt"
	"time"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/domain"
	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func MapTrip(row rowScanner) (domain.Trip, error) {
	var (
		id       uuid.UUID
		userID   uuid.UUID
		driverID uuid.UUID

		startLatitude  float64
		startLongitude float64
		endLatitude    float64
		endLongitude   float64

		priceRaw  int64
		statusRaw string

		startedAt  time.Time
		finishedAt *time.Time
		createdAt  time.Time
		updatedAt  time.Time
	)

	if err := row.Scan(
		&id,
		&userID,
		&driverID,
		&startLatitude,
		&startLongitude,
		&endLatitude,
		&endLongitude,
		&priceRaw,
		&statusRaw,
		&startedAt,
		&finishedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.Trip{}, err
	}

	startPoint, err := vobject.NewCoordinates(
		startLatitude,
		startLongitude,
	)
	if err != nil {
		return domain.Trip{}, fmt.Errorf(
			"restore start point: %w",
			err,
		)
	}

	endPoint, err := vobject.NewCoordinates(
		endLatitude,
		endLongitude,
	)
	if err != nil {
		return domain.Trip{}, fmt.Errorf(
			"restore end point: %w",
			err,
		)
	}

	price, err := vobject.NewMoney(priceRaw)
	if err != nil {
		return domain.Trip{}, fmt.Errorf(
			"restore price: %w",
			err,
		)
	}

	status, err := vobject.NewTripStatus(statusRaw)
	if err != nil {
		return domain.Trip{}, fmt.Errorf(
			"restore status: %w",
			err,
		)
	}

	return domain.Trip{
		ID:         id,
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: startPoint,
		EndPoint:   endPoint,
		Price:      price,
		Status:     status,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}, nil
}
