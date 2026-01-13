package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kodra-pay/transaction-service/internal/clients"
	"github.com/kodra-pay/transaction-service/internal/config"
	"github.com/kodra-pay/transaction-service/internal/handlers"
	"github.com/kodra-pay/transaction-service/internal/queue"
	"github.com/kodra-pay/transaction-service/internal/repositories"
	"github.com/kodra-pay/transaction-service/internal/services"
)

func Register(app *fiber.App, serviceName string) {
	health := handlers.NewHealthHandler(serviceName)
	health.Register(app)

	cfg := config.Load(serviceName, "7004")

	// Initialize Transaction Repository
	repo, err := repositories.NewTransactionRepository(cfg.PostgresDSN)
	if err != nil {
		panic(err)
	}

	// Initialize Fee Repository (shares DB connection)
	feeRepo := repositories.NewTransactionFeeRepository(repo.GetDB())

	// Initialize Subscription Client
	subClient := clients.NewSubscriptionClient(cfg.SubscriptionServiceURL)

	// Initialize settlement event publisher
	publisher := queue.NewSettlementPublisher()

	// Initialize Transaction Service V2 (with fees)
	svc := services.NewTransactionServiceV2(repo, feeRepo, publisher, subClient)
	handler := handlers.NewTransactionHandler(svc)

	app.Get("/transactions", handler.List)
	app.Post("/transactions", handler.Create)
	app.Get("/transactions/:reference", handler.Get)
	app.Post("/transactions/:reference/capture", handler.Capture)
	app.Post("/transactions/:reference/refund", handler.Refund)
	app.Put("/transactions/:reference/status", handler.UpdateStatus)
}
