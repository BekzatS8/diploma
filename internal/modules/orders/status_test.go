package orders

import "testing"

func TestOrderStatusTransitions(t *testing.T) {
	allowed := [][2]string{
		{StatusDraft, StatusPaymentPending},
		{StatusDraft, StatusPublished},
		{StatusDraft, StatusCancelled},
		{StatusPaymentPending, StatusPublished},
		{StatusPaymentPending, StatusDraft},
		{StatusPaymentPending, StatusCancelled},
		{StatusPublished, StatusInProgress},
		{StatusPublished, StatusCancelled},
		{StatusInProgress, StatusCompleted},
		{StatusCompleted, StatusInProgress},
	}
	for _, tc := range allowed {
		if !CanTransition(tc[0], tc[1]) {
			t.Fatalf("CanTransition(%q, %q) = false, want true", tc[0], tc[1])
		}
	}

	rejected := [][2]string{
		{StatusPublished, StatusDraft},
		{StatusInProgress, StatusCancelled},
		{StatusCompleted, StatusCancelled},
		{StatusCancelled, StatusDraft},
	}
	for _, tc := range rejected {
		if CanTransition(tc[0], tc[1]) {
			t.Fatalf("CanTransition(%q, %q) = true, want false", tc[0], tc[1])
		}
	}
}

func TestOrderCancelAndDeleteGuards(t *testing.T) {
	selected := "executor-id"
	if CanCancel(StatusPublished, &selected) {
		t.Fatal("CanCancel published order with selected executor = true, want false")
	}
	if !CanCancel(StatusPublished, nil) {
		t.Fatal("CanCancel published order without selected executor = false, want true")
	}
	if !CanDelete(StatusDraft) || !CanDelete(StatusCancelled) {
		t.Fatal("CanDelete must allow draft and cancelled orders")
	}
	if CanDelete(StatusPublished) || CanDelete(StatusInProgress) || CanDelete(StatusCompleted) {
		t.Fatal("CanDelete must reject active orders")
	}
}
