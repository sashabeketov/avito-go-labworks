package http

import (
	"context"

	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
)

type ReadinessChecker interface {
	CheckAll(ctx context.Context) error
}

type Handler struct {
	api.Unimplemented

	readiness ReadinessChecker
}

func NewHandler(readiness ReadinessChecker) *Handler {
	return &Handler{
		readiness: readiness,
	}
}

var _ api.ServerInterface = (*Handler)(nil)
