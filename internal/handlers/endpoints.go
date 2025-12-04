package handlers

import (
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
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.MerchantID == "" || req.Amount <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "merchant_id and positive amount are required")
	}

	resp, err := h.svc.Create(c.Context(), req)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create transaction")
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *TransactionHandler) Get(c *fiber.Ctx) error {
	ref := c.Params("reference")
	if ref == "" {
		return fiber.NewError(fiber.StatusBadRequest, "reference is required")
	}
	resp, err := h.svc.Get(c.Context(), ref)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch transaction")
	}
	if resp.Reference == "" {
		return fiber.NewError(fiber.StatusNotFound, "transaction not found")
	}
	return c.JSON(resp)
}

func (h *TransactionHandler) List(c *fiber.Ctx) error {
	merchantID := c.Query("merchant_id")
	if merchantID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "merchant_id is required")
	}
	resp, err := h.svc.ListByMerchant(c.Context(), merchantID, 50)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list transactions")
	}
	return c.JSON(resp)
}

func (h *TransactionHandler) Capture(c *fiber.Ctx) error {
	ref := c.Params("reference")
	return c.JSON(h.svc.Capture(c.Context(), ref))
}

func (h *TransactionHandler) Refund(c *fiber.Ctx) error {
	ref := c.Params("reference")
	return c.JSON(h.svc.Refund(c.Context(), ref))
}
