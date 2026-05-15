package wallets

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) Ensure(ctx context.Context, userID, currency string) (Wallet, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO wallets(user_id, balance, currency)
		VALUES($1, 0, $2)
		ON CONFLICT (user_id) DO UPDATE SET updated_at=wallets.updated_at
		RETURNING user_id, balance::float8, currency, updated_at
	`, userID, currency)
	return scanWallet(row)
}

func (r *Repository) Credit(ctx context.Context, userID, actorID string, amount float64, currency, reason string) (Wallet, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Wallet{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO wallets(user_id, balance, currency)
		VALUES($1, 0, $2)
		ON CONFLICT (user_id) DO NOTHING
	`, userID, currency); err != nil {
		return Wallet{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE wallets SET balance=balance+$2, currency=$3, updated_at=NOW()
		WHERE user_id=$1
	`, userID, amount, currency); err != nil {
		return Wallet{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO wallet_transactions(user_id, amount, direction, currency, reason, created_by)
		VALUES($1, $2, 'credit', $3, $4, $5)
	`, userID, amount, currency, reason, actorID); err != nil {
		return Wallet{}, err
	}
	row := tx.QueryRow(ctx, `SELECT user_id, balance::float8, currency, updated_at FROM wallets WHERE user_id=$1`, userID)
	wallet, err := scanWallet(row)
	if err != nil {
		return Wallet{}, err
	}
	return wallet, tx.Commit(ctx)
}

func (r *Repository) ListTransactions(ctx context.Context, userID string, limit int) ([]Transaction, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, user_id::text, amount::float8, direction, currency, reason, order_id::text, created_by::text, created_at
		FROM wallet_transactions
		WHERE user_id=$1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Transaction, 0)
	for rows.Next() {
		var item Transaction
		if err := rows.Scan(&item.ID, &item.UserID, &item.Amount, &item.Direction, &item.Currency, &item.Reason, &item.OrderID, &item.CreatedBy, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanWallet(row pgx.Row) (Wallet, error) {
	var item Wallet
	err := row.Scan(&item.UserID, &item.Balance, &item.Currency, &item.UpdatedAt)
	return item, err
}
