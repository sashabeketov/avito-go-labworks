package service

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/sashabeketov/avito-go-labworks/internal/domain"
	"github.com/sashabeketov/avito-go-labworks/internal/domain/vobject"
	"github.com/sashabeketov/avito-go-labworks/internal/port"
)

type CreateTripInput struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint vobject.Coordinates
	EndPoint   vobject.Coordinates
	Price      int64
}

type TripService struct {
	tripRepository    port.TripRepository
	historyRepository port.TripStatusHistoryRepository
	txManager         port.TxManager
}

func NewTripService(
	tripRepository port.TripRepository,
	historyRepository port.TripStatusHistoryRepository,
	txManager port.TxManager,
) *TripService {
	return &TripService{
		tripRepository:    tripRepository,
		historyRepository: historyRepository,
		txManager:         txManager,
	}
}

func (s *TripService) Create(
	ctx context.Context,
	input CreateTripInput,
) (domain.Trip, error) {
	if input.UserID == uuid.Nil() {
		return domain.Trip{}, fmt.Errorf("user ID is empty: %w", domain.ErrInvalidTrip)
	}
	if input.DriverID == uuid.Nil() {
		return domain.Trip{}, fmt.Errorf("driver ID is empty: %w", domain.ErrInvalidTrip)
	}

	startPoint, err := vobject.NewCoordinates(
		input.StartPoint.Latitude,
		input.StartPoint.Longitude,
	)
	if err != nil {
		return domain.Trip{}, fmt.Errorf(
			"invalid start point: %w: %w",
			domain.ErrInvalidTrip,
			err,
		)
	}

	endPoint, err := vobject.NewCoordinates(
		input.EndPoint.Latitude,
		input.EndPoint.Longitude,
	)
	if err != nil {
		return domain.Trip{}, fmt.Errorf(
			"invalid end point: %w: %w",
			domain.ErrInvalidTrip,
			err,
		)
	}

	price, err := vobject.NewMoney(input.Price)
	if err != nil {
		return domain.Trip{}, fmt.Errorf(
			"invalid price: %w: %w",
			domain.ErrInvalidTrip,
			err,
		)
	}

	now := time.Now().UTC()
	trip := domain.Trip{
		ID:         uuid.New(),
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		StartPoint: startPoint,
		EndPoint:   endPoint,
		Price:      price,
		Status:     vobject.TripStatusActive,
		StartedAt:  now,
	}

	var createdTrip domain.Trip
	err = s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.tripRepository.Create(txCtx, trip); err != nil {
			return fmt.Errorf("insert trip: %w", err)
		}

		history := domain.TripStatusHistory{
			TripID:    trip.ID,
			ToStatus:  vobject.TripStatusActive,
			ChangedAt: now,
		}
		if err := s.historyRepository.Create(txCtx, history); err != nil {
			return fmt.Errorf("insert trip status history: %w", err)
		}

		var err error
		createdTrip, err = s.tripRepository.GetByID(txCtx, trip.ID)
		if err != nil {
			return fmt.Errorf("read created trip: %w", err)
		}

		return nil
	})
	if err != nil {
		return domain.Trip{}, fmt.Errorf("create trip: %w", err)
	}

	return createdTrip, nil
}

func (s *TripService) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.Trip, error) {
	trip, err := s.tripRepository.GetByID(ctx, id)
	if err != nil {
		return domain.Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return trip, nil
}

func (s *TripService) Finish(
	ctx context.Context,
	id uuid.UUID,
) (domain.Trip, error) {
	var finishedTrip domain.Trip
	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		finishedAt := time.Now().UTC()
		trip, err := s.tripRepository.Finish(txCtx, id, finishedAt)
		if err != nil {
			return fmt.Errorf("update trip: %w", err)
		}

		fromStatus := vobject.TripStatusActive
		history := domain.TripStatusHistory{
			TripID:     id,
			FromStatus: &fromStatus,
			ToStatus:   vobject.TripStatusCompleted,
			ChangedAt:  finishedAt,
		}
		if err := s.historyRepository.Create(txCtx, history); err != nil {
			return fmt.Errorf("insert trip status history: %w", err)
		}

		finishedTrip = trip
		return nil
	})
	if err != nil {
		return domain.Trip{}, fmt.Errorf("finish trip: %w", err)
	}

	return finishedTrip, nil
}
