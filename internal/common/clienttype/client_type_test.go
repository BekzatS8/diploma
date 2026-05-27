package clienttype

import "testing"

func TestNormalizeAndValidate(t *testing.T) {
	if got := Normalize("  TOO  "); got != "too" {
		t.Fatalf("normalize: got %q", got)
	}
	if err := Validate("too"); err != nil {
		t.Fatalf("too: %v", err)
	}
	if err := Validate("company"); err == nil {
		t.Fatal("expected error for company")
	}
}
