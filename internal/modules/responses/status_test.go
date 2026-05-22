package responses

import "testing"

func TestResponseStatusTransitions(t *testing.T) {
	allowed := [][2]string{
		{StatusDraft, StatusPaymentPending},
		{StatusDraft, StatusCancelled},
		{StatusPaymentPending, StatusSubmitted},
		{StatusPaymentPending, StatusDraft},
		{StatusPaymentPending, StatusCancelled},
		{StatusSubmitted, StatusAccepted},
		{StatusSubmitted, StatusRejected},
	}
	for _, tc := range allowed {
		if !CanTransition(tc[0], tc[1]) {
			t.Fatalf("CanTransition(%q, %q) = false, want true", tc[0], tc[1])
		}
	}

	rejected := [][2]string{
		{StatusDraft, StatusSubmitted},
		{StatusSubmitted, StatusDraft},
		{StatusAccepted, StatusRejected},
		{StatusRejected, StatusAccepted},
		{StatusCancelled, StatusDraft},
	}
	for _, tc := range rejected {
		if CanTransition(tc[0], tc[1]) {
			t.Fatalf("CanTransition(%q, %q) = true, want false", tc[0], tc[1])
		}
	}
}

func TestResponseVisibilityAndMutationGuards(t *testing.T) {
	if !CanCancel(StatusDraft) || !CanCancel(StatusPaymentPending) {
		t.Fatal("CanCancel must allow draft and payment_pending responses")
	}
	if CanCancel(StatusSubmitted) || CanCancel(StatusAccepted) || CanCancel(StatusRejected) {
		t.Fatal("CanCancel must reject submitted/accepted/rejected responses")
	}
	if !CanDelete(StatusDraft) || !CanDelete(StatusCancelled) {
		t.Fatal("CanDelete must allow draft and cancelled responses")
	}
	if !IsClientVisible(StatusSubmitted, true) || !IsClientVisible(StatusAccepted, true) {
		t.Fatal("IsClientVisible must allow paid submitted and accepted responses")
	}
	if IsClientVisible(StatusRejected, true) || IsClientVisible(StatusSubmitted, false) {
		t.Fatal("IsClientVisible must hide rejected or unpaid responses")
	}
}
