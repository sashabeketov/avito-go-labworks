package readiness

import (
	"context"
	"fmt"
)

type Readiness struct {
	checkers []Checker
}

func CreateReadinessCheckers(checkers ...Checker) *Readiness {
	return &Readiness{
		checkers: checkers,
	}
}

// CheckAll возможно добавить сохранание данных о конкретных узнал отедльно
func (r *Readiness) CheckAll(ctx context.Context) error {
	for _, checker := range r.checkers {
		if err := checker.Check(ctx); err != nil {
			return fmt.Errorf(
				"%s readiness check failed: %w",
				checker.Name(),
				err,
			)
		}
	}

	return nil
}
