//go:build integration

package orders

import (
	"context"
	"os"
	"testing"

	"buhpro/internal/config"
	"buhpro/internal/platform/db"
)

func TestSubmitWithPaymentIntegration(t *testing.T) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		t.Skip("DB_URL not set")
	}

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	cfg.DB.URL = dbURL
	pool, err := db.NewPool(ctx, cfg.DB)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	repo := NewRepository(pool)
	// Use existing draft order from audit if present — create minimal via SQL
	var orderID, clientID string
	err = pool.QueryRow(ctx, `
		SELECT o.id, o.client_id FROM orders o
		WHERE o.status = 'draft' AND o.deleted_at IS NULL
		ORDER BY o.created_at DESC LIMIT 1
	`).Scan(&orderID, &clientID)
	if err != nil {
		t.Fatalf("find draft order: %v", err)
	}

	_, _, err = repo.SubmitWithPayment(ctx, orderID, clientID, 1000, 0, 1000, 2000, "KZT", "mock", "mock_ref_test", "https://mock/checkout")
	if err != nil {
		t.Fatalf("SubmitWithPayment: %v", err)
	}
}
