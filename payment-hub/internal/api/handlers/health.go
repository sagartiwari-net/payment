package handlers

import (
	"context"
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct {
	db  *sql.DB
	env string
}

func NewHealthHandler(db *sql.DB, env string) *HealthHandler {
	return &HealthHandler{db: db, env: env}
}

func (h *HealthHandler) Health(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "ok"
	if err := h.db.PingContext(ctx); err != nil {
		dbStatus = "error"
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":   "unhealthy",
			"service":  "payment-hub",
			"env":      h.env,
			"database": dbStatus,
		})
	}

	return c.JSON(fiber.Map{
		"status":   "ok",
		"service":  "payment-hub",
		"env":      h.env,
		"database": dbStatus,
	})
}
