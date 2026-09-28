package port

import "context"

type TxManager interface {
	Do(ctx context.Context, fn func(ctx2 context.Context) error) error
}
