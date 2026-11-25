package health

import "context"

type readinessChecker interface {
	CheckAll(ctx context.Context) error
}
