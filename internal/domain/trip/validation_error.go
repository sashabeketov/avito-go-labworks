package trip

import "fmt"

type ValidationError struct {
	Field  string
	Reason string
	Cause  error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Reason)
}

func (e *ValidationError) Unwrap() []error {
	return []error{ErrInvalidTrip, e.Cause}
}
