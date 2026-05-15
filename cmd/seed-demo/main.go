package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"buhpro/internal/config"
	authmodule "buhpro/internal/modules/auth"
	"buhpro/internal/platform/db"

	"github.com/jackc/pgx/v5"
)

const (
	demoClientID   = "00000000-0000-0000-0000-000000000101"
	demoExecutorID = "00000000-0000-0000-0000-000000000102"
	demoCoachID    = "00000000-0000-0000-0000-000000000103"
	demoAdminID    = "00000000-0000-0000-0000-000000000104"

	categoryTaxID   = 90001
	categoryAuditID = 90002

	courseID      = "00000000-0000-0000-0000-000000000201"
	assignmentID  = "00000000-0000-0000-0000-000000000202"
	draftOrderID  = "00000000-0000-0000-0000-000000000301"
	publicOrderID = "00000000-0000-0000-0000-000000000302"

	draftResponseID     = "00000000-0000-0000-0000-000000000401"
	submittedResponseID = "00000000-0000-0000-0000-000000000402"

	sanctionID = "00000000-0000-0000-0000-000000000501"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if strings.EqualFold(cfg.App.Env, "production") {
		log.Fatal("demo seed is disabled in production environment")
	}
	if !isDemoSeedEnabled() {
		log.Fatal("demo seed is disabled; set ENABLE_DEMO_SEED=true")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DB)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	if err := seed(ctx, pool); err != nil {
		log.Fatalf("seed failed: %v", err)
	}

	log.Println("demo seed completed")
}

func isDemoSeedEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("ENABLE_DEMO_SEED")))
	return value == "1" || value == "true" || value == "yes"
}

func seed(ctx context.Context, pool dbPool) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	demoPassword := os.Getenv("DEMO_USER_PASSWORD")
	if strings.TrimSpace(demoPassword) == "" {
		demoPassword = "DemoPass123"
	}
	hash, err := authmodule.HashPassword(demoPassword)
	if err != nil {
		return err
	}

	if err := upsertUser(ctx, tx, demoClientID, "demo.client@buhpro.local", hash, "client"); err != nil {
		return err
	}
	if err := upsertUser(ctx, tx, demoExecutorID, "demo.executor@buhpro.local", hash, "executor"); err != nil {
		return err
	}
	if err := upsertUser(ctx, tx, demoCoachID, "demo.coach@buhpro.local", hash, "coach"); err != nil {
		return err
	}
	if err := upsertUser(ctx, tx, demoAdminID, "demo.admin@buhpro.local", hash, "admin"); err != nil {
		return err
	}

	if err := upsertProfiles(ctx, tx); err != nil {
		return err
	}
	if err := upsertCategories(ctx, tx); err != nil {
		return err
	}
	if err := upsertCourse(ctx, tx); err != nil {
		return err
	}
	if err := upsertOrders(ctx, tx); err != nil {
		return err
	}
	if err := upsertResponses(ctx, tx); err != nil {
		return err
	}
	if err := upsertSanction(ctx, tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

type dbPool interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

func upsertUser(ctx context.Context, tx pgx.Tx, id, email, hash, role string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO users(id, email, password_hash, role, is_active, verification_status)
		VALUES($1,$2,$3,$4,TRUE,'verified')
		ON CONFLICT (id) DO UPDATE
		SET email=EXCLUDED.email,
			password_hash=EXCLUDED.password_hash,
			role=EXCLUDED.role,
			is_active=TRUE,
			verification_status='verified',
			updated_at=NOW()
	`, id, email, hash, role)
	return err
}

func upsertProfiles(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `INSERT INTO client_profiles(user_id, company_name, about) VALUES($1,$2,$3) ON CONFLICT (user_id) DO UPDATE SET company_name=EXCLUDED.company_name, about=EXCLUDED.about, updated_at=NOW()`, demoClientID, "Demo Client LLC", "Demo client profile"); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO executor_profiles(user_id, display_name, bio, years_experience, rating_avg, rating_count, completed_orders, sanction_points, verification_status, verified_at) VALUES($1,$2,$3,3,4.50,2,1,0,'verified',NOW()) ON CONFLICT (user_id) DO UPDATE SET display_name=EXCLUDED.display_name, bio=EXCLUDED.bio, years_experience=EXCLUDED.years_experience, verification_status='verified', verified_at=COALESCE(executor_profiles.verified_at, NOW()), updated_at=NOW()`, demoExecutorID, "Demo Executor", "Handles bookkeeping tasks"); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO coach_profiles(user_id, display_name, bio, expertise) VALUES($1,$2,$3,$4) ON CONFLICT (user_id) DO UPDATE SET display_name=EXCLUDED.display_name, bio=EXCLUDED.bio, expertise=EXCLUDED.expertise, updated_at=NOW()`, demoCoachID, "Demo Coach", "Finance educator", "Tax and compliance"); err != nil {
		return err
	}
	return nil
}

func upsertCategories(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `INSERT INTO categories(id, slug, name, is_active) VALUES($1,'tax','Tax Services',TRUE) ON CONFLICT (slug) DO UPDATE SET name=EXCLUDED.name, is_active=TRUE`, categoryTaxID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO categories(id, slug, name, is_active) VALUES($1,'audit','Audit',TRUE) ON CONFLICT (slug) DO UPDATE SET name=EXCLUDED.name, is_active=TRUE`, categoryAuditID); err != nil {
		return err
	}
	return nil
}

func upsertCourse(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO courses(id, coach_id, category_id, title, description, is_published)
		VALUES($1,$2,$3,'Demo Tax Compliance 101','Published demo course for sanctions follow-up',TRUE)
		ON CONFLICT (id) DO UPDATE
		SET title=EXCLUDED.title, description=EXCLUDED.description, is_published=TRUE, updated_at=NOW(), deleted_at=NULL
	`, courseID, demoCoachID, categoryTaxID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO course_assignments(id, course_id, executor_id, sanction_id, assigned_by, reason, source, status)
		VALUES($1,$2,$3,NULL,$4,'Demo assignment for executor','manual_admin','assigned')
		ON CONFLICT (id) DO UPDATE
		SET course_id=EXCLUDED.course_id, executor_id=EXCLUDED.executor_id, assigned_by=EXCLUDED.assigned_by, reason=EXCLUDED.reason, source=EXCLUDED.source, status='assigned', updated_at=NOW(), completed_at=NULL
	`, assignmentID, courseID, demoExecutorID, demoAdminID); err != nil {
		return err
	}
	return nil
}

func upsertOrders(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO orders(id, client_id, category_id, title, description, budget_amount, currency, status)
		VALUES($1,$2,$3,'Demo draft order','Draft order for demo flow',45000,'KZT','draft')
		ON CONFLICT (id) DO UPDATE
		SET title=EXCLUDED.title, description=EXCLUDED.description, budget_amount=EXCLUDED.budget_amount, status='draft', selected_executor_id=NULL, selected_response_id=NULL, published_at=NULL, completed_at=NULL, cancelled_at=NULL, updated_at=NOW(), deleted_at=NULL
	`, draftOrderID, demoClientID, categoryTaxID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO orders(id, client_id, category_id, title, description, budget_amount, currency, status, published_at)
		VALUES($1,$2,$3,'Demo published order','Published order for response demo',80000,'KZT','published',NOW())
		ON CONFLICT (id) DO UPDATE
		SET title=EXCLUDED.title, description=EXCLUDED.description, budget_amount=EXCLUDED.budget_amount, status='published', published_at=COALESCE(orders.published_at, NOW()), selected_executor_id=NULL, selected_response_id=NULL, completed_at=NULL, cancelled_at=NULL, updated_at=NOW(), deleted_at=NULL
	`, publicOrderID, demoClientID, categoryAuditID); err != nil {
		return err
	}
	return nil
}

func upsertResponses(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO responses(id, order_id, executor_id, cover_letter, proposed_amount, currency, status, is_paid)
		VALUES($1,$2,$3,'Draft demo response',30000,'KZT','draft',FALSE)
		ON CONFLICT (id) DO UPDATE
		SET cover_letter=EXCLUDED.cover_letter, proposed_amount=EXCLUDED.proposed_amount, status='draft', is_paid=FALSE, paid_at=NULL, updated_at=NOW(), deleted_at=NULL
	`, draftResponseID, draftOrderID, demoExecutorID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO responses(id, order_id, executor_id, cover_letter, proposed_amount, currency, status, is_paid, paid_at)
		VALUES($1,$2,$3,'Submitted demo response',70000,'KZT','submitted',TRUE,NOW())
		ON CONFLICT (id) DO UPDATE
		SET cover_letter=EXCLUDED.cover_letter, proposed_amount=EXCLUDED.proposed_amount, status='submitted', is_paid=TRUE, paid_at=COALESCE(responses.paid_at, NOW()), updated_at=NOW(), deleted_at=NULL
	`, submittedResponseID, publicOrderID, demoExecutorID); err != nil {
		return err
	}
	return nil
}

func upsertSanction(ctx context.Context, tx pgx.Tx) error {
	enable := strings.EqualFold(strings.TrimSpace(os.Getenv("DEMO_INCLUDE_SANCTION")), "true")
	if !enable {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO sanctions(id, executor_id, status, reason, severity, issued_by, started_at, ends_at, metadata)
		VALUES($1,$2,'active','low_rating_first',2,$3,NOW(),NOW()+INTERVAL '7 days', jsonb_build_object('seed',true,'source','demo_seed'))
		ON CONFLICT (id) DO UPDATE
		SET status='active', reason='low_rating_first', severity=2, issued_by=$3, started_at=NOW(), ends_at=NOW()+INTERVAL '7 days', metadata=jsonb_build_object('seed',true,'source','demo_seed')
	`, sanctionID, demoExecutorID, demoAdminID)
	return err
}
