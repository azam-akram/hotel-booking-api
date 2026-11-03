package model

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrRoomUnavailable     = errors.New("room is not available for the requested dates")
	ErrOverCapacity        = errors.New("guests exceed room capacity")
	ErrInvalidInput        = errors.New("invalid input")
	ErrPayloadTooLarge     = errors.New("request body too large")
	ErrIdempotencyMismatch = errors.New("idempotency key was used with a different request")
	ErrDuplicateReference  = errors.New("booking reference already exists")
	ErrDuplicateKey        = errors.New("idempotency key already exists")
)

const (
	CodeInvalidInput        = "invalid_input"
	CodeNotFound            = "not_found"
	CodeMethodNotAllowed    = "method_not_allowed"
	CodePayloadTooLarge     = "payload_too_large"
	CodeRoomUnavailable     = "room_unavailable"
	CodeOverCapacity        = "over_capacity"
	CodeIdempotencyMismatch = "idempotency_mismatch"
	CodeInternal            = "internal_error"
)

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Invalidf returns an error that wraps ErrInvalidInput with a formatted detail.
func Invalidf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, fmt.Sprintf(format, args...))
}
