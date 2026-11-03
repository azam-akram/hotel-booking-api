package model

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidInput = errors.New("invalid input")
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
