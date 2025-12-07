package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/kodra-pay/transaction-service/internal/models"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(dsn string) (*TransactionRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return &TransactionRepository{db: db}, nil
}

func (r *TransactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	query := `
		INSERT INTO transactions (reference, merchant_id, customer_email, customer_id, customer_name, amount, currency, status, payment_method, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	if err := r.db.QueryRowContext(ctx, query,
		tx.Reference, tx.MerchantID, tx.CustomerEmail, tx.CustomerID, tx.CustomerName,
		tx.Amount, tx.Currency, tx.Status, tx.PaymentMethod, tx.Description,
	).Scan(&tx.ID, &tx.CreatedAt, &tx.UpdatedAt); err != nil {
		return err
	}

	// Record ledger credit for this merchant to feed settlement calculations.
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO wallet_ledger (
			merchant_id, transaction_id, entry_type, amount, balance_after,
			currency, description, reference, created_at
		)
		VALUES (
			$1,
			$2,
			'credit',
			$3,
			(SELECT COALESCE(MAX(balance_after), 0) + $3 FROM wallet_ledger WHERE merchant_id = $1),
			$4,
			$5,
			$6,
			NOW()
		)
	`, tx.MerchantID, tx.ID, tx.Amount, tx.Currency, "Transaction credit", tx.Reference)
	if err != nil {
		// Log the error but don't fail the transaction creation
		fmt.Printf("failed to record ledger entry: %v\n", err)
	}

	return nil
}

func (r *TransactionRepository) GetByReference(ctx context.Context, reference int) (*models.Transaction, error) {
	query := `
		SELECT id, reference, merchant_id, customer_email, customer_id, customer_name, amount, currency, status, payment_method, description, created_at, updated_at
		FROM transactions
		WHERE reference = $1
	`
	var tx models.Transaction
	err := r.db.QueryRowContext(ctx, query, reference).Scan(
		&tx.ID, &tx.Reference, &tx.MerchantID, &tx.CustomerEmail, &tx.CustomerID, &tx.CustomerName,
		&tx.Amount, &tx.Currency, &tx.Status, &tx.PaymentMethod, &tx.Description,
		&tx.CreatedAt, &tx.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &tx, err
}

func (r *TransactionRepository) ListByMerchant(ctx context.Context, merchantID int, limit int) ([]*models.Transaction, error) {
	query := `
		SELECT id, reference, merchant_id, customer_email, customer_id, customer_name, amount, currency, status, payment_method, description, created_at, updated_at
		FROM transactions
		WHERE merchant_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, merchantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.Transaction
	for rows.Next() {
		var tx models.Transaction
		if err := rows.Scan(
			&tx.ID, &tx.Reference, &tx.MerchantID, &tx.CustomerEmail, &tx.CustomerID, &tx.CustomerName,
			&tx.Amount, &tx.Currency, &tx.Status, &tx.PaymentMethod, &tx.Description,
			&tx.CreatedAt, &tx.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &tx)
	}
	return list, rows.Err()
}
