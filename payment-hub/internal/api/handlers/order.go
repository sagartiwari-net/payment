package handlers

import (
	"errors"

	"github.com/buyahref/payment-hub/internal/api/middleware"
	"github.com/buyahref/payment-hub/internal/models"
	"github.com/buyahref/payment-hub/internal/repository"
	"github.com/buyahref/payment-hub/internal/services"
	"github.com/gofiber/fiber/v2"
)

type OrderHandler struct {
	orders *services.OrderService
}

func NewOrderHandler(orders *services.OrderService) *OrderHandler {
	return &OrderHandler{orders: orders}
}

type createOrderRequest struct {
	OrderID    string              `json:"order_id"`
	Amount     float64             `json:"amount"`
	Currency   string              `json:"currency"`
	Customer   models.CustomerInput `json:"customer"`
	Product    models.ProductInput  `json:"product"`
	ReturnURL  string              `json:"return_url"`
	WebhookURL string              `json:"webhook_url"`
}

func (h *OrderHandler) Create(c *fiber.Ctx) error {
	merchant := middleware.GetMerchant(c)
	if merchant == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "error": "unauthorized"})
	}

	var req createOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid json body",
		})
	}

	result, err := h.orders.Create(c.Context(), merchant, models.CreateOrderInput{
		OrderID:    req.OrderID,
		Amount:     req.Amount,
		Currency:   req.Currency,
		Customer:   req.Customer,
		Product:    req.Product,
		ReturnURL:  req.ReturnURL,
		WebhookURL: req.WebhookURL,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidInput):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
		case errors.Is(err, services.ErrDuplicateOrder):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"success": false, "error": "duplicate order_id"})
		case errors.Is(err, services.ErrMerchantSuspended):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "error": "merchant suspended"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "failed to create order"})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"hub_order_id": result.HubOrderID,
			"payment_url":  result.PaymentURL,
			"expires_at":   result.ExpiresAt.Format("2006-01-02T15:04:05Z"),
		},
	})
}

func (h *OrderHandler) Verify(c *fiber.Ctx) error {
	merchant := middleware.GetMerchant(c)
	if merchant == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "error": "unauthorized"})
	}

	orderID := c.Params("order_id")
	if orderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "order_id required"})
	}

	result, err := h.orders.Verify(c.Context(), merchant, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": "order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "failed to verify order"})
	}

	data := fiber.Map{
		"hub_order_id": result.HubOrderID,
		"order_id":     result.OrderID,
		"status":       result.Status,
		"amount":       result.Amount,
		"currency":     result.Currency,
	}
	if result.PhonePeTxnID != "" {
		data["phonepe_txn_id"] = result.PhonePeTxnID
	}
	if result.PaidAt != nil {
		data["paid_at"] = result.PaidAt.Format("2006-01-02T15:04:05Z")
	}

	return c.JSON(fiber.Map{"success": true, "data": data})
}
