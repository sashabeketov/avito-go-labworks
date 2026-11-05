package readiness

import "context"

type Checker interface {
	Name() string
	Check(ctx context.Context) error
}
