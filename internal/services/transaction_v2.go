package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kodra-pay/transaction-service/internal/clients"
	"github.com/kodra-pay/transaction-service/internal/dto"
	"github.com/kodra-pay/transaction-service/internal/models"
	"github.com/kodra-pay/transaction-service/internal/queue"
	"github.com/kodra-pay/transaction-service/internal/repositories"
)

const PLATFORM_MERCHANT_ID = 999999 // Platform revenue account (created in migration)

type TransactionServiceV2 struct {
	repo                *repositories.TransactionRepository
	feeRepo             *repositories.TransactionFeeRepository
	settlementPublisher *queue.SettlementPublisher
	subscriptionClient  clients.SubscriptionClient
}

func NewTransactionServiceV2(
	repo *repositories.TransactionRepository,
	feeRepo *repositories.TransactionFeeRepository,
	publisher *queue.SettlementPublisher,
	subscriptionClient clients.SubscriptionClient,
) *TransactionServiceV2 {
	return &TransactionServiceV2{
		repo:                repo,
		feeRepo:             feeRepo,
		settlementPublisher: publisher,
		subscriptionClient:  subscriptionClient,
	}
}

func (s *TransactionServiceV2) Create(ctx context.Context, req dto.TransactionCreateRequest) (dto.TransactionResponse, error) {
	ref := req.Reference
	if ref == "" {
		// Generate reference if not provided
	}

	email := req.CustomerEmail
	paymentMethod := req.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "card"
	}

	status := req.Status
	if status == "" {
		status = "success"
	}

	amountKobo := int64(math.Round(req.Amount * 100))

	tx := &models.Transaction{
		Reference:     ref,
		MerchantID:    req.MerchantID,
		CustomerEmail: email,
		CustomerID:    req.CustomerID,
		CustomerName:  req.CustomerName,
		Amount:        amountKobo,
		Currency:      req.Currency,
		Status:        status,
		PaymentMethod: paymentMethod,
		Description:   req.Description,
	}

	// IDEMPOTENCY CHECK: Prevent duplicate charges
	// If transaction with this reference already exists, return it instead of creating duplicate
	if ref != "" {
		existingTx, err := s.repo.GetByReference(ctx, ref)
		if err == nil && existingTx != nil {
			log.Printf("INFO: Idempotency check - transaction with reference %s already exists (ID: %d), returning existing", ref, existingTx.ID)
			return dto.TransactionResponse{
				ID:            existingTx.ID,
				Reference:     existingTx.Reference,
				MerchantID:    existingTx.MerchantID,
				CustomerEmail: existingTx.CustomerEmail,
				CustomerID:    existingTx.CustomerID,
				CustomerName:  existingTx.CustomerName,
				Amount:        float64(existingTx.Amount) / 100,
				Currency:      existingTx.Currency,
				Status:        existingTx.Status,
				Description:   existingTx.Description,
				CreatedAt:     existingTx.CreatedAt,
			}, nil
		}
	}

	if err := s.repo.Create(ctx, tx); err != nil {
		return dto.TransactionResponse{}, err
	}

	// Calculate and deduct fees for revenue-generating transactions
	if tx.Status != "payout" && tx.PaymentMethod != "payout" && s.subscriptionClient != nil {
		go s.processFeeAndBalance(ctx, tx)
	}

	// Publish settlement event to Redis queue
	if s.settlementPublisher != nil && tx.Status != "payout" && tx.PaymentMethod != "payout" {
		go func() {
			publishCtx := context.Background()
			if err := s.settlementPublisher.PublishTransaction(publishCtx, tx.MerchantID, tx.Amount, tx.Currency, tx.ID); err != nil {
				log.Printf("Failed to publish settlement event: %v\n", err)
			}
		}()
	}

	return dto.TransactionResponse{
		ID:            tx.ID,
		Reference:     tx.Reference,
		MerchantID:    tx.MerchantID,
		CustomerEmail: tx.CustomerEmail,
		CustomerID:    tx.CustomerID,
		CustomerName:  tx.CustomerName,
		Amount:        float64(tx.Amount) / 100,
		Currency:      tx.Currency,
		Status:        tx.Status,
		Description:   tx.Description,
		CreatedAt:     tx.CreatedAt,
	}, nil
}

// processFeeAndBalance calculates fees and updates merchant + platform balances
func (s *TransactionServiceV2) processFeeAndBalance(ctx context.Context, tx *models.Transaction) {
	// Step 1: Calculate fee using subscription service
	feeQuote, err := s.subscriptionClient.CalculateFee(ctx, &clients.FeeQuoteRequest{
		MerchantID:    int64(tx.MerchantID),
		Amount:        float64(tx.Amount) / 100, // Convert kobo to naira
		PaymentMethod: tx.PaymentMethod,
		Currency:      tx.Currency,
	})

	if err != nil {
		log.Printf("ERROR: Failed to calculate fee for transaction %d: %v", tx.ID, err)
		// Fallback: Update merchant with full amount (no fee deduction)
		s.updateMerchantBalance(tx.MerchantID, tx.Currency, float64(tx.Amount)/100)
		return
	}

	// Step 2: Record fee in transaction_fees table
	if s.feeRepo != nil {
		fee := &repositories.TransactionFee{
			TransactionID:     int64(tx.ID),
			MerchantID:        tx.MerchantID,
			TransactionAmount: tx.Amount,
			PaymentMethod:     tx.PaymentMethod,
			Currency:          tx.Currency,
			FeePercentage:     feeQuote.FeePercentage,
			FeeFlat:           feeQuote.FeeFlat,
			FeeCap:            feeQuote.FeeCap,
			TotalFee:          feeQuote.TotalFee,
			FeeCapped:         feeQuote.FeeCapped,
			PlatformRevenue:   feeQuote.TotalFee, // Platform earns the fee
			MerchantNet:       feeQuote.MerchantNet,
			UsedCustomPricing: feeQuote.UsedCustomPricing,
			CreatedAt:         time.Now(),
		}

		if err := s.feeRepo.RecordFee(context.Background(), fee); err != nil {
			log.Printf("ERROR: Failed to record fee for transaction %d: %v", tx.ID, err)
		} else {
			log.Printf("✅ Recorded fee for transaction %d: Platform earns ₦%.2f, Merchant gets ₦%.2f",
				tx.ID, float64(feeQuote.TotalFee)/100, float64(feeQuote.MerchantNet)/100)
		}
	}

	// Step 3: Update merchant balance with NET amount (after fees)
	merchantNetAmount := float64(feeQuote.MerchantNet) / 100 // Convert kobo to naira
	s.updateMerchantBalance(tx.MerchantID, tx.Currency, merchantNetAmount)
	log.Printf("✅ Updated merchant %d balance: +₦%.2f (net after fees)", tx.MerchantID, merchantNetAmount)

	// Step 4: Update platform revenue account with FEE amount
	platformFeeAmount := float64(feeQuote.TotalFee) / 100 // Convert kobo to naira
	s.updateMerchantBalance(PLATFORM_MERCHANT_ID, tx.Currency, platformFeeAmount)
	log.Printf("💰 Updated platform revenue: +₦%.2f (fee from transaction %d)", platformFeeAmount, tx.ID)
}

// updateMerchantBalance calls the merchant service to update the balance
func (s *TransactionServiceV2) updateMerchantBalance(merchantID int, currency string, amount float64) {
	merchantServiceURL := os.Getenv("MERCHANT_SERVICE_URL")
	if merchantServiceURL == "" {
		merchantServiceURL = "http://merchant-service:7002"
	}

	url := fmt.Sprintf("%s/internal/balance/record", merchantServiceURL)
	payload := map[string]interface{}{
		"merchant_id": merchantID,
		"currency":    currency,
		"amount":      amount,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Warning: failed to call merchant service to update balance: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("Warning: merchant service returned non-ok status for balance update: %d, body: %s\n", resp.StatusCode, respBody)
	}
}

func (s *TransactionServiceV2) Get(ctx context.Context, reference string) (dto.TransactionResponse, error) {
	tx, err := s.repo.GetByReference(ctx, reference)
	if err != nil {
		return dto.TransactionResponse{}, err
	}
	if tx == nil {
		return dto.TransactionResponse{}, nil
	}
	return dto.TransactionResponse{
		ID:            tx.ID,
		Reference:     tx.Reference,
		MerchantID:    tx.MerchantID,
		CustomerEmail: tx.CustomerEmail,
		CustomerID:    tx.CustomerID,
		CustomerName:  tx.CustomerName,
		Amount:        float64(tx.Amount) / 100.0,
		Currency:      tx.Currency,
		Status:        tx.Status,
		Description:   tx.Description,
		CreatedAt:     tx.CreatedAt,
	}, nil
}

func (s *TransactionServiceV2) Capture(ctx context.Context, reference string) dto.TransactionResponse {
	_ = ctx
	return dto.TransactionResponse{Reference: reference, Status: "captured"}
}

func (s *TransactionServiceV2) Refund(ctx context.Context, reference string) dto.TransactionResponse {
	_ = ctx
	return dto.TransactionResponse{Reference: reference, Status: "refunded"}
}

func (s *TransactionServiceV2) ListByMerchant(ctx context.Context, merchantID int, limit int) (dto.TransactionListResponse, error) {
	list, err := s.repo.ListByMerchant(ctx, merchantID, limit)
	if err != nil {
		return dto.TransactionListResponse{}, err
	}
	res := dto.TransactionListResponse{}
	for _, tx := range list {
		res.Transactions = append(res.Transactions, dto.TransactionResponse{
			ID:            tx.ID,
			Reference:     tx.Reference,
			MerchantID:    tx.MerchantID,
			CustomerEmail: tx.CustomerEmail,
			CustomerID:    tx.CustomerID,
			CustomerName:  tx.CustomerName,
			Amount:        float64(tx.Amount) / 100,
			Currency:      tx.Currency,
			Status:        tx.Status,
			Description:   tx.Description,
			CreatedAt:     tx.CreatedAt,
		})
	}
	return res, nil
}

func (s *TransactionServiceV2) ListByStatus(ctx context.Context, status string, limit int) (dto.TransactionListResponse, error) {
	list, err := s.repo.ListByStatus(ctx, status, limit)
	if err != nil {
		return dto.TransactionListResponse{}, err
	}
	res := dto.TransactionListResponse{}
	for _, tx := range list {
		res.Transactions = append(res.Transactions, dto.TransactionResponse{
			ID:            tx.ID,
			Reference:     tx.Reference,
			MerchantID:    tx.MerchantID,
			CustomerEmail: tx.CustomerEmail,
			CustomerID:    tx.CustomerID,
			CustomerName:  tx.CustomerName,
			Amount:        float64(tx.Amount) / 100,
			Currency:      tx.Currency,
			Status:        tx.Status,
			Description:   tx.Description,
			CreatedAt:     tx.CreatedAt,
		})
	}
	return res, nil
}

func (s *TransactionServiceV2) UpdateStatus(ctx context.Context, reference, status string) error {
	statusLower := strings.ToLower(status)
	switch statusLower {
	case "success", "successful", "pending", "pending_review", "failed", "payout":
	default:
		return fmt.Errorf("invalid status")
	}
	return s.repo.UpdateStatusByReference(ctx, reference, statusLower)
}
