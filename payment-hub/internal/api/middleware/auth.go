package middleware

import (
	"strconv"
	"time"

	"github.com/buyahref/payment-hub/internal/models"
	"github.com/buyahref/payment-hub/internal/repository"
	"github.com/buyahref/payment-hub/internal/security"
	"github.com/gofiber/fiber/v2"
)

const (
	headerMerchantKey = "X-Merchant-Key"
	headerTimestamp   = "X-Timestamp"
	headerSignature   = "X-Signature"

	merchantLocalKey = "merchant"
)

func MerchantAuth(merchants *repository.MerchantRepository, maxSkew time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get(headerMerchantKey)
		timestamp := c.Get(headerTimestamp)
		signature := c.Get(headerSignature)

		if apiKey == "" || timestamp == "" || signature == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "missing authentication headers",
			})
		}

		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "invalid timestamp",
			})
		}

		now := time.Now().UTC().Unix()
		if abs(now-ts) > int64(maxSkew.Seconds()) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "timestamp expired",
			})
		}

		merchant, err := merchants.GetByAPIKey(c.Context(), apiKey)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "invalid merchant key",
			})
		}

		body := string(c.Body())
		if !security.Verify(merchant.APISecret, timestamp, c.Method(), c.Path(), body, signature) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "invalid signature",
			})
		}

		c.Locals(merchantLocalKey, merchant)
		return c.Next()
	}
}

func GetMerchant(c *fiber.Ctx) *models.Merchant {
	m, _ := c.Locals(merchantLocalKey).(*models.Merchant)
	return m
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
