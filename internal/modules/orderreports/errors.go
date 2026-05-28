package orderreports

import "errors"

var (
	ErrNotFound          = errors.New("report not found")
	ErrOrderNotFound     = errors.New("order not found")
	ErrOrderNotReportable = errors.New("order cannot be reported")
	ErrDuplicatePending  = errors.New("pending report already exists")
	ErrInvalidInput      = errors.New("invalid input")
	ErrAlreadyReviewed   = errors.New("report already reviewed")
)
