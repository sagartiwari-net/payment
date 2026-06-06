package api

import (
	"database/sql"
	"time"

	"github.com/buyahref/payment-hub/internal/api/handlers"
	"github.com/buyahref/payment-hub/internal/api/middleware"
	"github.com/buyahref/payment-hub/internal/config"
	"github.com/buyahref/payment-hub/internal/phonepe"
	"github.com/buyahref/payment-hub/internal/repository"
	"github.com/buyahref/payment-hub/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"
)

func NewApp(cfg *config.Config, log *zap.Logger, db *sql.DB) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Buyahref Payment Hub",
		ServerHeader: "PaymentHub",
	})

	app.Use(recover.New())
	app.Use(middleware.NormalizePath())
	app.Use(middleware.RequestLogger(log))

	merchantRepo := repository.NewMerchantRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	orderService := services.NewOrderService(orderRepo, cfg.AppURL, cfg.OrderExpiryMinutes)
	ppClient := phonepe.NewClient(cfg.PhonePeClientConfig())
	paymentService := services.NewPaymentService(orderRepo, merchantRepo, ppClient, cfg.AppURL, log)

	healthHandler := handlers.NewHealthHandler(db, cfg.AppEnv)
	orderHandler := handlers.NewOrderHandler(orderService)
	checkoutHandler := handlers.NewCheckoutHandler(paymentService)
	webhookHandler := handlers.NewPhonePeWebhookHandler(paymentService)

	app.Get("/health", healthHandler.Health)

	// Public checkout + PhonePe
	app.Get("/pay/:token", checkoutHandler.Show)
	app.Post("/pay/:token/pay", checkoutHandler.Pay)
	app.All("/pay/:token/return", checkoutHandler.Return)
	app.Post("/webhooks/phonepe", webhookHandler.Handle)

	maxSkew := time.Duration(cfg.SignatureMaxAgeMinutes) * time.Minute
	rateLimiter := middleware.NewRateLimiter(100, 20, time.Minute)

	api := app.Group("/api/v1")
	api.Use(rateLimiter.Middleware())
	api.Use(middleware.MerchantAuth(merchantRepo, maxSkew))

	api.Post("/orders/create", orderHandler.Create)
	api.Get("/orders/:order_id/verify", orderHandler.Verify)

	return app
}
