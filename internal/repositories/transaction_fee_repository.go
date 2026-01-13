package repositories

import (
	"context"
	"database/sql"
	"time"
)

type TransactionFeeRepository struct {
	db *sql.DB
}

type TransactionFee struct {
	ID                  int64
	TransactionID       int64
	MerchantID          int
	TransactionAmount   int64
	PaymentMethod       string
	Currency            string
	FeePercentage       float64
	FeeFlat             int64
	FeeCap              *int64
	TotalFee            int64
	FeeCapped           bool
	PlatformRevenue     int64
	MerchantNet         int64
	UsedCustomPricing   bool
	CreatedAt           time.Time
}

func NewTransactionFeeRepository(db *sql.DB) *TransactionFeeRepository {
	return &TransactionFeeRepository{db: db}
}

func (r *TransactionFeeRepository) RecordFee(ctx context.Context, fee *TransactionFee) error {
	query := `
		INSERT INTO transaction_fees (
			transaction_id, merchant_id, transaction_amount, payment_method, currency,
			fee_percentage, fee_flat, fee_cap, total_fee, fee_capped,
			platform_revenue, merchant_net, used_custom_pricing, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14
		) RETURNING id
	`

	err := r.db.QueryRowContext(
		ctx, query,
		fee.TransactionID, fee.MerchantID, fee.TransactionAmount, fee.PaymentMethod, fee.Currency,
		fee.FeePercentage, fee.FeeFlat, fee.FeeCap, fee.TotalFee, fee.FeeCapped,
		fee.PlatformRevenue, fee.MerchantNet, fee.UsedCustomPricing, fee.CreatedAt,
	).Scan(&fee.ID)

	return err
}

func (r *TransactionFeeRepository) GetByTransactionID(ctx context.Context, transactionID int64) (*TransactionFee, error) {
	query := `
		SELECT id, transaction_id, merchant_id, transaction_amount, payment_method, currency,
		       fee_percentage, fee_flat, fee_cap, total_fee, fee_capped,
		       platform_revenue, merchant_net, used_custom_pricing, created_at
		FROM transaction_fees
		WHERE transaction_id = $1
	`

	fee := &TransactionFee{}
	err := r.db.QueryRowContext(ctx, query, transactionID).Scan(
		&fee.ID, &fee.TransactionID, &fee.MerchantID, &fee.TransactionAmount, &fee.PaymentMethod, &fee.Currency,
		&fee.FeePercentage, &fee.FeeFlat, &fee.FeeCap, &fee.TotalFee, &fee.FeeCapped,
		&fee.PlatformRevenue, &fee.MerchantNet, &fee.UsedCustomPricing, &fee.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return fee, err
}
