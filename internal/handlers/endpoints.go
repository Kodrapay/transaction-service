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
	return c.JSON(h.svc.Create(c.Context(), req))
}

func (h *TransactionHandler) Get(c *fiber.Ctx) error {
	ref := c.Params("reference")
	return c.JSON(h.svc.Get(c.Context(), ref))
}

func (h *TransactionHandler) List(c *fiber.Ctx) error {
	merchantID := c.Query("merchant_id")
	return c.JSON(h.svc.ListByMerchant(c.Context(), merchantID, 50))
}

func (h *TransactionHandler) Capture(c *fiber.Ctx) error {
	ref := c.Params("reference")
	return c.JSON(h.svc.Capture(c.Context(), ref))
}

func (h *TransactionHandler) Refund(c *fiber.Ctx) error {
	ref := c.Params("reference")
	return c.JSON(h.svc.Refund(c.Context(), ref))
}
