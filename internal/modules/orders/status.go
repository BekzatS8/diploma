package orders

import "strings"

const (
	StatusDraft          = "draft"
	StatusPaymentPending = "payment_pending"
	StatusPublished      = "published"
	StatusInProgress     = "in_progress"
	StatusCompleted      = "completed"
	StatusCancelled      = "cancelled"
)

func IsKnownStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case StatusDraft, StatusPaymentPending, StatusPublished, StatusInProgress, StatusCompleted, StatusCancelled:
		return true
	default:
		return false
	}
}

func CanTransition(from, to string) bool {
	switch from {
	case StatusDraft:
		return to == StatusPaymentPending || to == StatusCancelled
	case StatusPaymentPending:
		return to == StatusPublished || to == StatusDraft || to == StatusCancelled
	case StatusPublished:
		return to == StatusInProgress || to == StatusCancelled
	case StatusInProgress:
		return to == StatusCompleted
	case StatusCompleted:
		return to == StatusInProgress
	default:
		return false
	}
}

func CanCancel(status string, selectedExecutorID *string) bool {
	if selectedExecutorID != nil {
		return false
	}
	return CanTransition(status, StatusCancelled)
}

func CanDelete(status string) bool {
	return status == StatusDraft || status == StatusCancelled
}
