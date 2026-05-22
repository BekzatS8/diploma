package orders

import (
	"net/url"
	"testing"
)

func TestParsePublicOrdersQuery(t *testing.T) {
	values := url.Values{
		"category":        {" TAX "},
		"budget_min":      {"1000"},
		"budget_max":      {"2000"},
		"deadline_before": {"2026-06-01T00:00:00Z"},
		"region":          {"Almaty"},
		"q":               {" payroll "},
		"page":            {"2"},
		"page_size":       {"50"},
	}

	q, message, ok := parsePublicOrdersQuery(values)
	if !ok {
		t.Fatalf("parsePublicOrdersQuery failed: %s", message)
	}
	if q.CategorySlug != "tax" || q.Region != "Almaty" || q.Q != "payroll" {
		t.Fatalf("unexpected normalized filters: %#v", q)
	}
	if q.BudgetMin == nil || *q.BudgetMin != 1000 || q.BudgetMax == nil || *q.BudgetMax != 2000 {
		t.Fatalf("unexpected budget filters: %#v", q)
	}
	if q.DeadlineBefore == nil {
		t.Fatal("deadline_before was not parsed")
	}
	if q.Page != 2 || q.PageSize != 50 {
		t.Fatalf("unexpected pagination: page=%d page_size=%d", q.Page, q.PageSize)
	}
}

func TestParsePublicOrdersQueryRejectsInvalidFilters(t *testing.T) {
	cases := []url.Values{
		{"budget_min": {"-1"}},
		{"budget_min": {"2000"}, "budget_max": {"1000"}},
		{"deadline_before": {"2026-06-01"}},
		{"page": {"0"}},
		{"page_size": {"101"}},
		{"q": {string(make([]byte, 201))}},
	}
	for _, values := range cases {
		if _, _, ok := parsePublicOrdersQuery(values); ok {
			t.Fatalf("parsePublicOrdersQuery(%v) succeeded, want failure", values)
		}
	}
}
