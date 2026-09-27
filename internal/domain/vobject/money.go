package vobject

import "errors"

type Money int64

func NewMoney(value int64) (Money, error) {
	if value < 0 {
		return 0, errors.New("money must be non-negative")
	}

	return Money(value), nil
}
