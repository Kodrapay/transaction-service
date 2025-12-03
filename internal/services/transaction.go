package services

import (
	"context"

	"github.com/google/uuid"

	"github.com/kodra-pay/transaction-service/internal/dto"
	"github.com/kodra-pay/transaction-service/internal/repositories"
)

type TransactionService struct {
	repo *repositories.TransactionRepository
}

func NewTransactionService(repo *repositories.TransactionRepository) *TransactionService {
	return &TransactionService{repo: repo}
}

func (s *TransactionService) Create(_ context.Context, req dto.TransactionCreateRequest) dto.TransactionResponse {
	return dto.TransactionResponse{Reference: "txn_" + uuid.NewString(), Status: "pending"}
}

func (s *TransactionService) Get(_ context.Context, reference string) dto.TransactionResponse {
	return dto.TransactionResponse{Reference: reference, Status: "pending"}
}

func (s *TransactionService) Capture(_ context.Context, reference string) dto.TransactionResponse {
	return dto.TransactionResponse{Reference: reference, Status: "captured"}
}

func (s *TransactionService) Refund(_ context.Context, reference string) dto.TransactionResponse {
	return dto.TransactionResponse{Reference: reference, Status: "refunded"}
}
