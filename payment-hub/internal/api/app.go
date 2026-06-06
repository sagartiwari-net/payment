package api

import (
	"database/sql"

	"github.com/buyahref/payment-hub/internal/api/handlers"
	"github.com/buyahref/payment-hub/internal/api/middleware"
	"github.com/buyahref/payment-hub/internal/config"
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

	healthHandler := handlers.NewHealthHandler(db, cfg.AppEnv)

	app.Get("/health", healthHandler.Health)

	return app
}
