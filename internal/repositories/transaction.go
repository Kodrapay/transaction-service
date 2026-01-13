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

func (r *TransactionRepository) GetDB() *sql.DB {
	return r.db
}

func (r *TransactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	// Start SQL transaction for atomicity
	dbTx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer dbTx.Rollback() // Will be no-op if committed

	// Insert transaction record
	query := `
		INSERT INTO transactions (reference, merchant_id, customer_email, customer_id, customer_name, amount, currency, status, payment_method, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, reference, created_at, updated_at
	`
	if err := dbTx.QueryRowContext(ctx, query,
		tx.Reference, tx.MerchantID, tx.CustomerEmail, tx.CustomerID, tx.CustomerName,
		tx.Amount, tx.Currency, tx.Status, tx.PaymentMethod, tx.Description,
	).Scan(&tx.ID, &tx.Reference, &tx.CreatedAt, &tx.UpdatedAt); err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}

	// Record ledger credit - skip for payout transactions
	if tx.Status != "payout" && tx.PaymentMethod != "payout" {
		// Lock the merchant's ledger rows to prevent race conditions
		// Calculate new balance atomically using window function
		ledgerQuery := `
			INSERT INTO wallet_ledger (
				merchant_id, transaction_id, entry_type, amount, balance_before, balance_after,
				currency, description, reference, created_at
			)
			SELECT 
				$1::bigint as merchant_id,
				$2::bigint as transaction_id,
				'credit' as entry_type,
				$3::bigint as amount,
				COALESCE((
					SELECT balance_after 
					FROM wallet_ledger 
					WHERE merchant_id = $1 
					ORDER BY id DESC 
					LIMIT 1
					FOR UPDATE  -- Lock row to prevent concurrent reads
				), 0) as balance_before,
				COALESCE((
					SELECT balance_after 
					FROM wallet_ledger 
					WHERE merchant_id = $1 
					ORDER BY id DESC 
					LIMIT 1
					FOR UPDATE
				), 0) + $3 as balance_after,
				$4::varchar as currency,
				$5::text as description,
				$6::varchar as reference,
				NOW() as created_at
		`
		if _, err := dbTx.ExecContext(ctx, ledgerQuery,
			tx.MerchantID, tx.ID, tx.Amount, tx.Currency, "Transaction credit", tx.Reference); err != nil {
			return fmt.Errorf("insert ledger entry: %w", err)
		}
	}

	// Commit transaction - both operations succeed or both fail
	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *TransactionRepository) GetByReference(ctx context.Context, reference string) (*models.Transaction, error) {
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

func (r *TransactionRepository) ListByStatus(ctx context.Context, status string, limit int) ([]*models.Transaction, error) {
	query := `
		SELECT id, reference, merchant_id, customer_email, customer_id, customer_name, amount, currency, status, payment_method, description, created_at, updated_at
		FROM transactions
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, status, limit)
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

// UpdateStatusByReference updates the status of a transaction by reference.
func (r *TransactionRepository) UpdateStatusByReference(ctx context.Context, reference, status string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE transactions
		SET status = $2, updated_at = NOW()
		WHERE reference = $1
	`, reference, status)
	return err
}
