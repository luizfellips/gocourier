package apperrors

import "errors"

var (
	ErrDuplicate   = errors.New("duplicate idempotency key")
	ErrNotFound    = errors.New("not found")
	ErrValidation  = errors.New("validation error")
	ErrTransient   = errors.New("transient error")
	ErrPermanent   = errors.New("permanent error")
	ErrUnauthorized = errors.New("unauthorized")
	ErrCircuitOpen = errors.New("circuit breaker open")
)

func IsTransient(err error) bool {
	return errors.Is(err, ErrTransient)
}

func IsPermanent(err error) bool {
	return errors.Is(err, ErrPermanent)
}
