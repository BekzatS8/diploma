package attachments

import "errors"

var (
	ErrNotFound     = errors.New("attachment not found")
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid input")
)
