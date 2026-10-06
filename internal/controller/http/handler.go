package http

import (
	"context"
	"uuid"

	"github.com/go-playground/validator/v10"
	"github.com/sashabeketov/avito-go-labworks/internal/domain"
	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
	"github.com/sashabeketov/avito-go-labworks/internal/service"
)

type TripService interface {
	Create(ctx context.Context, input service.CreateTripInput) (domain.Trip, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Trip, error)
	Finish(ctx context.Context, id uuid.UUID) (domain.Trip, error)
}

type ReadinessChecker interface {
	CheckAll(ctx context.Context) error
}

var validate *validator.Validate

type Handler struct {
	api.Unimplemented

	readiness ReadinessChecker

	tripService TripService
}

func NewHandler(
	readiness ReadinessChecker,
	tripService TripService,
) *Handler {
	validate = validator.New()
	return &Handler{
		readiness:   readiness,
		tripService: tripService,
	}
}

var _ api.ServerInterface = (*Handler)(nil)
