package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kodra-pay/transaction-service/internal/handlers"
	"github.com/kodra-pay/transaction-service/internal/config"
	"github.com/kodra-pay/transaction-service/internal/repositories"
	"github.com/kodra-pay/transaction-service/internal/services"
)

func Register(app *fiber.App, serviceName string) {
	health := handlers.NewHealthHandler(serviceName)
	health.Register(app)

	cfg := config.Load(serviceName, "7004")

	// Try to initialize repository, but don't panic if it fails
	repo, err := repositories.NewTransactionRepository(cfg.PostgresDSN)
	if err != nil {
		// Log error but continue with empty handler
		println("Warning: Failed to connect to database:", err.Error())
		// Return empty array for now
		app.Get("/transactions", func(c *fiber.Ctx) error {
			return c.JSON([]interface{}{})
		})
		return
	}

	svc := services.NewTransactionService(repo)
	handler := handlers.NewTransactionHandler(svc)

	app.Get("/transactions", handler.List)
	app.Post("/transactions", handler.Create)
	app.Get("/transactions/:reference", handler.Get)
	app.Post("/transactions/:reference/capture", handler.Capture)
	app.Post("/transactions/:reference/refund", handler.Refund)
}
