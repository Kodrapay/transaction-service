package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kodra-pay/transaction-service/internal/dto"
	"github.com/kodra-pay/transaction-service/internal/repositories"
	"github.com/kodra-pay/transaction-service/internal/models"
)

type TransactionService struct {
	repo *repositories.TransactionRepository
}

func NewTransactionService(repo *repositories.TransactionRepository) *TransactionService {
	return &TransactionService{repo: repo}
}

func (s *TransactionService) Create(ctx context.Context, req dto.TransactionCreateRequest) dto.TransactionResponse {
	ref := "txn_" + uuid.NewString()
	tx := &models.Transaction{
		Reference:    ref,
		MerchantID:   req.MerchantID,
		CustomerEmail: req.CustomerID,
		Amount:       req.Amount,
		Currency:     req.Currency,
		Status:       "pending",
		Description:  req.Description,
	}
	_ = s.repo.Create(ctx, tx)
	return dto.TransactionResponse{
		ID:          tx.ID,
		Reference:   tx.Reference,
		MerchantID:  tx.MerchantID,
		CustomerEmail: tx.CustomerEmail,
		CustomerName: "",
		Amount:      tx.Amount,
		Currency:    tx.Currency,
		Status:      tx.Status,
		Description: tx.Description,
		CreatedAt:   tx.CreatedAt.Format(time.RFC3339),
	}
}

func (s *TransactionService) Get(ctx context.Context, reference string) dto.TransactionResponse {
	tx, _ := s.repo.GetByReference(ctx, reference)
	if tx == nil {
		return dto.TransactionResponse{}
	}
	return dto.TransactionResponse{
		ID:          tx.ID,
		Reference:   tx.Reference,
		MerchantID:  tx.MerchantID,
		CustomerEmail: tx.CustomerEmail,
		CustomerName: tx.CustomerName,
		Amount:      tx.Amount,
		Currency:    tx.Currency,
		Status:      tx.Status,
		Description: tx.Description,
		CreatedAt:   tx.CreatedAt.Format(time.RFC3339),
	}
}

func (s *TransactionService) Capture(ctx context.Context, reference string) dto.TransactionResponse {
	_ = ctx
	return dto.TransactionResponse{Reference: reference, Status: "captured"}
}

func (s *TransactionService) Refund(ctx context.Context, reference string) dto.TransactionResponse {
	_ = ctx
	return dto.TransactionResponse{Reference: reference, Status: "refunded"}
}

func (s *TransactionService) ListByMerchant(ctx context.Context, merchantID string, limit int) dto.TransactionListResponse {
	list, _ := s.repo.ListByMerchant(ctx, merchantID, limit)
	res := dto.TransactionListResponse{}
	for _, tx := range list {
		res.Transactions = append(res.Transactions, dto.TransactionResponse{
			ID:          tx.ID,
			Reference:   tx.Reference,
			MerchantID:  tx.MerchantID,
			CustomerEmail: tx.CustomerEmail,
			CustomerName: tx.CustomerName,
			Amount:      tx.Amount,
			Currency:    tx.Currency,
			Status:      tx.Status,
			Description: tx.Description,
			CreatedAt:   tx.CreatedAt.Format(time.RFC3339),
		})
	}
	return res
}
