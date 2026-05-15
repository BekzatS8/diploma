package leads

import "errors"

var (
	ErrNotFound         = errors.New("lead not found")
	ErrInvalidInput     = errors.New("invalid input")
	ErrInvalidStatus    = errors.New("invalid lead status")
	ErrDuplicate        = errors.New("duplicate lead")
	ErrEmailExists      = errors.New("email already exists")
	ErrAlreadyConverted = errors.New("lead already converted")
	ErrDocumentRequired = errors.New("required document is missing")
	ErrDocumentTooLarge = errors.New("document is too large")
	ErrInvalidMime      = errors.New("invalid document mime type")
)
