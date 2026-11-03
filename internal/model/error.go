package model

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
