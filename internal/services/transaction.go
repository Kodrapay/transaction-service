package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	ref := req.Reference // int. If 0, it means no reference was provided.
	if ref == 0 {
		// In a real scenario, an int reference would be generated, perhaps from a sequence or tx.ID after creation.
		// For now, we'll rely on it being provided or handled by the DB.
	}

	email := req.CustomerEmail
	// Assuming CustomerID is now the primary identifier. No need to fall back to email string.

	paymentMethod := req.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "card"
	}

	tx := &models.Transaction{
		Reference:     ref, // int
		MerchantID:    req.MerchantID, // int
		CustomerEmail: email,
		CustomerID:    req.CustomerID, // int
		CustomerName:  req.CustomerName,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Status:        "success",
		PaymentMethod: paymentMethod,
		Description:   req.Description,
	}

	if err := s.repo.Create(ctx, tx); err != nil {
		return dto.TransactionResponse{}, err
	}

	// Update merchant balance asynchronously
	go s.updateMerchantBalance(tx.MerchantID, tx.Currency, tx.Amount) // tx.MerchantID is int

	// Publish settlement event to Redis queue
	if s.settlementPublisher != nil {
		go func() {
			publishCtx := context.Background()
			if err := s.settlementPublisher.PublishTransaction(publishCtx, tx.MerchantID, tx.Amount, tx.Currency, tx.ID); err != nil { // tx.MerchantID, tx.ID are int
				// Log error but don't fail the transaction
				fmt.Printf("Failed to publish settlement event: %v\n", err)
			}
		}()
	}

	return dto.TransactionResponse{
		ID:            tx.ID, // int
		Reference:     tx.Reference, // int
		MerchantID:    tx.MerchantID, // int
		CustomerEmail: tx.CustomerEmail,
		CustomerID:    tx.CustomerID, // int
		CustomerName:  tx.CustomerName,
		Amount:        tx.Amount,
		Currency:      tx.Currency,
		Status:        tx.Status,
		Description:   tx.Description,
		CreatedAt:     tx.CreatedAt,
	}, nil
}

// updateMerchantBalance calls the merchant service to update the balance
func (s *TransactionService) updateMerchantBalance(merchantID int, currency string, amount int64) { // int
	merchantServiceURL := os.Getenv("MERCHANT_SERVICE_URL")
	if merchantServiceURL == "" {
		merchantServiceURL = "http://merchant-service:7002"
	}

	url := fmt.Sprintf("%s/internal/balance/record", merchantServiceURL)
	payload := map[string]interface{}{
		"merchant_id": merchantID, // int
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
	_, _ = client.Do(req)
	// Ignore errors - balance update is not critical for transaction success
}

func (s *TransactionService) Get(ctx context.Context, reference int) (dto.TransactionResponse, error) { // int
	tx, err := s.repo.GetByReference(ctx, reference) // int
	if err != nil {
		return dto.TransactionResponse{}, err
	}
	if tx == nil {
		return dto.TransactionResponse{}, nil
	}
	return dto.TransactionResponse{
		ID:            tx.ID, // int
		Reference:     tx.Reference, // int
		MerchantID:    tx.MerchantID, // int
		CustomerEmail: tx.CustomerEmail,
		CustomerID:    tx.CustomerID, // int
		CustomerName:  tx.CustomerName,
		Amount:        tx.Amount,
		Currency:      tx.Currency,
		Status:        tx.Status,
		Description:   tx.Description,
		CreatedAt:     tx.CreatedAt,
	}, nil
}

func (s *TransactionService) Capture(ctx context.Context, reference int) dto.TransactionResponse { // int
	_ = ctx
	return dto.TransactionResponse{Reference: reference, Status: "captured"}
}

func (s *TransactionService) Refund(ctx context.Context, reference int) dto.TransactionResponse { // int
	_ = ctx
	return dto.TransactionResponse{Reference: reference, Status: "refunded"}
}

func (s *TransactionService) ListByMerchant(ctx context.Context, merchantID int, limit int) (dto.TransactionListResponse, error) { // int
	list, err := s.repo.ListByMerchant(ctx, merchantID, limit) // int
	if err != nil {
		return dto.TransactionListResponse{}, err
	}
	res := dto.TransactionListResponse{}
	for _, tx := range list {
		res.Transactions = append(res.Transactions, dto.TransactionResponse{
			ID:            tx.ID, // int
			Reference:     tx.Reference, // int
			MerchantID:    tx.MerchantID, // int
			CustomerEmail: tx.CustomerEmail,
			CustomerID:    tx.CustomerID, // int
			CustomerName:  tx.CustomerName,
			Amount:        tx.Amount,
			Currency:      tx.Currency,
			Status:        tx.Status,
			Description:   tx.Description,
			CreatedAt:     tx.CreatedAt,
		})
	}
	return res, nil
}
