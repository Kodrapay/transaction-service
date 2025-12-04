package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kodra-pay/transaction-service/internal/dto"
	"github.com/kodra-pay/transaction-service/internal/models"
	"github.com/kodra-pay/transaction-service/internal/repositories"
)

type TransactionService struct {
	repo *repositories.TransactionRepository
}

func NewTransactionService(repo *repositories.TransactionRepository) *TransactionService {
	return &TransactionService{repo: repo}
}

func (s *TransactionService) Create(ctx context.Context, req dto.TransactionCreateRequest) (dto.TransactionResponse, error) {
	ref := req.Reference
	if ref == "" {
		ref = "txn_" + uuid.NewString()
	}

	email := req.CustomerEmail
	if email == "" {
		email = req.CustomerID
	}

	paymentMethod := req.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "card"
	}

	tx := &models.Transaction{
		Reference:     ref,
		MerchantID:    req.MerchantID,
		CustomerEmail: email,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Status:        "success",
		PaymentMethod: paymentMethod,
		Description:   req.Description,
	}

	if err := s.repo.Create(ctx, tx); err != nil {
		return dto.TransactionResponse{}, err
	}

	return dto.TransactionResponse{
		ID:            tx.ID,
		Reference:     tx.Reference,
		MerchantID:    tx.MerchantID,
		CustomerEmail: tx.CustomerEmail,
		CustomerName:  tx.CustomerName,
		Amount:        tx.Amount,
		Currency:      tx.Currency,
		Status:        tx.Status,
		Description:   tx.Description,
		CreatedAt:     tx.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *TransactionService) Get(ctx context.Context, reference string) (dto.TransactionResponse, error) {
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
		CustomerName:  tx.CustomerName,
		Amount:        tx.Amount,
		Currency:      tx.Currency,
		Status:        tx.Status,
		Description:   tx.Description,
		CreatedAt:     tx.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *TransactionService) Capture(ctx context.Context, reference string) dto.TransactionResponse {
	_ = ctx
	return dto.TransactionResponse{Reference: reference, Status: "captured"}
}

func (s *TransactionService) Refund(ctx context.Context, reference string) dto.TransactionResponse {
	_ = ctx
	return dto.TransactionResponse{Reference: reference, Status: "refunded"}
}

func (s *TransactionService) ListByMerchant(ctx context.Context, merchantID string, limit int) (dto.TransactionListResponse, error) {
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
			CustomerName:  tx.CustomerName,
			Amount:        tx.Amount,
			Currency:      tx.Currency,
			Status:        tx.Status,
			Description:   tx.Description,
			CreatedAt:     tx.CreatedAt.Format(time.RFC3339),
		})
	}
	return res, nil
}
