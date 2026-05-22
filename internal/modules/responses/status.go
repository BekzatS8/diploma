package responses

import "strings"

const (
	StatusDraft          = "draft"
	StatusPaymentPending = "payment_pending"
	StatusSubmitted      = "submitted"
	StatusAccepted       = "accepted"
	StatusRejected       = "rejected"
	StatusCancelled      = "cancelled"
)

func IsKnownStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case StatusDraft, StatusPaymentPending, StatusSubmitted, StatusAccepted, StatusRejected, StatusCancelled:
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
		return to == StatusSubmitted || to == StatusDraft || to == StatusCancelled
	case StatusSubmitted:
		return to == StatusAccepted || to == StatusRejected
	default:
		return false
	}
}

func CanCancel(status string) bool {
	return CanTransition(status, StatusCancelled)
}

func CanDelete(status string) bool {
	return status == StatusDraft || status == StatusCancelled
}

func IsClientVisible(status string, isPaid bool) bool {
	return isPaid && (status == StatusSubmitted || status == StatusAccepted)
}
