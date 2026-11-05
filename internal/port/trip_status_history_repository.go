package port

import (
	"context"

	"github.com/sashabeketov/avito-go-labworks/internal/domain"
)

type TripStatusHistoryRepository interface {
	Create(ctx context.Context, info domain.TripStatusHistory) error
}
