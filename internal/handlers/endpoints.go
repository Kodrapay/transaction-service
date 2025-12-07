package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/kodra-pay/transaction-service/internal/dto"
	"github.com/kodra-pay/transaction-service/internal/services"
)

type TransactionHandler struct {
	svc *services.TransactionService
}

func NewTransactionHandler(svc *services.TransactionService) *TransactionHandler {
	return &TransactionHandler{svc: svc}
}

func (h *TransactionHandler) Create(c *fiber.Ctx) error {
	var req dto.TransactionCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
	}
	if req.MerchantID == 0 || req.Amount <= 0 { // int check
		return fiber.NewError(fiber.StatusBadRequest, "merchant_id and positive amount are required")
	}

	resp, err := h.svc.Create(c.Context(), req)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("failed to create transaction: %v", err))
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *TransactionHandler) Get(c *fiber.Ctx) error {
	ref := c.Params("reference") // Use c.Params
	if ref == "" {
		return fiber.NewError(fiber.StatusBadRequest, "reference is required")
	}
	resp, err := h.svc.Get(c.Context(), ref) // Pass string
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch transaction")
	}
	if resp.Reference == "" { // Check for empty string
		return fiber.NewError(fiber.StatusNotFound, "transaction not found")
	}
	return c.JSON(resp)
}

func (h *TransactionHandler) List(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50) // Default limit to 50

	status := c.Query("status")
	if status != "" {
		resp, err := h.svc.ListByStatus(c.Context(), status, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to list transactions by status")
		}
		return c.JSON(resp)
	}

	merchantID := c.QueryInt("merchant_id", 0)
	if merchantID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "merchant_id is required unless filtering by status")
	}
	resp, err := h.svc.ListByMerchant(c.Context(), merchantID, limit)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list transactions by merchant")
	}
	return c.JSON(resp)
}

func (h *TransactionHandler) Capture(c *fiber.Ctx) error {
	ref := c.Params("reference") // Use c.Params
	if ref == "" {
		return fiber.NewError(fiber.StatusBadRequest, "reference is required")
	}
	return c.JSON(h.svc.Capture(c.Context(), ref))
}

func (h *TransactionHandler) Refund(c *fiber.Ctx) error {
	ref := c.Params("reference") // Use c.Params
	if ref == "" {
		return fiber.NewError(fiber.StatusBadRequest, "reference is required")
	}
	return c.JSON(h.svc.Refund(c.Context(), ref))
}
