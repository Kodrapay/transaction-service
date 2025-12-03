package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kodra-pay/transaction-service/internal/config"
	"github.com/kodra-pay/transaction-service/internal/handlers"
	"github.com/kodra-pay/transaction-service/internal/repositories"
	"github.com/kodra-pay/transaction-service/internal/services"
)

func Register(app *fiber.App, cfg config.Config, repo *repositories.TransactionRepository) {
	health := handlers.NewHealthHandler(cfg.ServiceName)
	health.Register(app)

	svc := services.NewTransactionService(repo)
	h := handlers.NewTransactionHandler(svc)
	api := app.Group("/transactions")
	api.Post("/", h.Create)
	api.Get("/:reference", h.Get)
	api.Post("/:reference/capture", h.Capture)
	api.Post("/:reference/refund", h.Refund)
}
