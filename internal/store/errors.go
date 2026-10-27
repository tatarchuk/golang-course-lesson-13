package store

import "errors"

// Sentinel errors of the store. The gRPC server maps them to status codes (InvalidArgument,
// NotFound, AlreadyExists, FailedPrecondition), as the HTTP handlers of the earlier lessons
// mapped them to 422, 404, 409 and 422.
var (
	ErrValidation       = errors.New("validation error")
	ErrNotFound         = errors.New("album not found")
	ErrConflict         = errors.New("album with this title and label already exists")
	ErrInvalidReference = errors.New("unknown reference")
)
