package uploads

import "errors"

var (
	ErrNotFound     = errors.New("upload not found")
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid input")
	ErrFileTooLarge = errors.New("file too large")
	ErrInvalidMime  = errors.New("invalid mime type")
)
