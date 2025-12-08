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
	"time"

	"github.com/kodra-pay/transaction-service/internal/dto"
	"github.com/kodra-pay/transaction-service/internal/models"
	"github.com/kodra-pay/transaction-service/internal/queue"
	"github.com/kodra-pay/transaction-service/internal/repositories"
)

type TransactionService struct {
	repo                *repositories.TransactionRepository
	settlementPublisher *queue.SettlementPublisher
}

func NewTransactionService(repo *repositories.TransactionRepository, publisher *queue.SettlementPublisher) *TransactionService {
	return &TransactionService{
		repo:                repo,
		settlementPublisher: publisher,
	}
}

func (s *TransactionService) Create(ctx context.Context, req dto.TransactionCreateRequest) (dto.TransactionResponse, error) {
	ref := req.Reference // is now string
	if ref == "" {       // Check for empty string instead of 0
		// In a real scenario, an int reference would be generated, perhaps from a sequence or tx.ID after creation.
		// For now, we'll rely on it being provided or handled by the DB.
		// For string references, you might want to generate a UUID here if not provided.
	}

	email := req.CustomerEmail
	// Assuming CustomerID is now the primary identifier. No need to fall back to email string.

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
		Reference:     ref, // string
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

	if err := s.repo.Create(ctx, tx); err != nil {
		return dto.TransactionResponse{}, err
	}

	// Update merchant balance asynchronously for revenue-generating transactions only
	if tx.Status != "payout" && tx.PaymentMethod != "payout" {
		amountCurrency := float64(tx.Amount) / 100
		go s.updateMerchantBalance(tx.MerchantID, tx.Currency, amountCurrency)
	}

	// Publish settlement event to Redis queue
	if s.settlementPublisher != nil && tx.Status != "payout" && tx.PaymentMethod != "payout" {
		go func() {
			publishCtx := context.Background()
			// tx.Reference is now string, need to convert tx.ID to string if settlementPublisher expects string reference
			if err := s.settlementPublisher.PublishTransaction(publishCtx, tx.MerchantID, tx.Amount, tx.Currency, tx.ID); err != nil {
				// Log error but don't fail the transaction
				log.Printf("Failed to publish settlement event: %v\n", err)
			}
		}()
	}

	return dto.TransactionResponse{
		ID:            tx.ID,
		Reference:     tx.Reference, // string
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

// updateMerchantBalance calls the merchant service to update the balance
func (s *TransactionService) updateMerchantBalance(merchantID int, currency string, amount float64) {
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
		fmt.Printf("Warning: failed to call merchant service to update balance: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		fmt.Printf("Warning: merchant service returned non-ok status for balance update: %d, body: %s\n", resp.StatusCode, respBody)
	}
	// Ignore errors - balance update is not critical for transaction success
}

func (s *TransactionService) Get(ctx context.Context, reference string) (dto.TransactionResponse, error) { // changed reference to string
	tx, err := s.repo.GetByReference(ctx, reference) // pass string
	if err != nil {
		return dto.TransactionResponse{}, err
	}
	if tx == nil {
		return dto.TransactionResponse{}, nil
	}
	return dto.TransactionResponse{
		ID:            tx.ID,
		Reference:     tx.Reference, // string
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

func (s *TransactionService) Capture(ctx context.Context, reference string) dto.TransactionResponse { // changed reference to string
	_ = ctx
	return dto.TransactionResponse{Reference: reference, Status: "captured"}
}

func (s *TransactionService) Refund(ctx context.Context, reference string) dto.TransactionResponse { // changed reference to string
	_ = ctx
	return dto.TransactionResponse{Reference: reference, Status: "refunded"}
}

func (s *TransactionService) ListByMerchant(ctx context.Context, merchantID int, limit int) (dto.TransactionListResponse, error) {
	list, err := s.repo.ListByMerchant(ctx, merchantID, limit)
	if err != nil {
		return dto.TransactionListResponse{}, err
	}
	res := dto.TransactionListResponse{}
	for _, tx := range list {
		res.Transactions = append(res.Transactions, dto.TransactionResponse{
			ID:            tx.ID,
			Reference:     tx.Reference, // string
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

func (s *TransactionService) ListByStatus(ctx context.Context, status string, limit int) (dto.TransactionListResponse, error) {
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
