package handlers

import (
	"errors"

	"github.com/buyahref/payment-hub/internal/repository"
	"github.com/buyahref/payment-hub/internal/services"
	"github.com/gofiber/fiber/v2"
)

type CheckoutHandler struct {
	payments *services.PaymentService
}

func NewCheckoutHandler(payments *services.PaymentService) *CheckoutHandler {
	return &CheckoutHandler{payments: payments}
}

func (h *CheckoutHandler) Show(c *fiber.Ctx) error {
	token := c.Params("token")
	view, _, err := h.payments.GetCheckout(c.Context(), token)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).SendString("Payment link not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("Something went wrong")
	}

	html, err := services.RenderCheckoutHTML(view)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Render error")
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	c.Set("X-Frame-Options", "DENY")
	return c.SendString(html)
}

func (h *CheckoutHandler) Pay(c *fiber.Ctx) error {
	token := c.Params("token")
	payURL, err := h.payments.InitiatePayment(c.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return c.Status(fiber.StatusNotFound).SendString("Payment link not found")
		case errors.Is(err, services.ErrOrderExpired):
			return c.Status(fiber.StatusGone).SendString("Payment link expired")
		case errors.Is(err, services.ErrOrderNotPayable):
			return c.Redirect(h.payments.CheckoutURL(token), fiber.StatusTemporaryRedirect)
		default:
			return c.Status(fiber.StatusBadGateway).SendString("Unable to start PhonePe payment: " + err.Error())
		}
	}
	return c.Redirect(payURL, fiber.StatusTemporaryRedirect)
}

func (h *CheckoutHandler) Return(c *fiber.Ctx) error {
	token := c.Params("token")
	base64Resp := c.FormValue("response")
	if base64Resp == "" {
		base64Resp = c.Query("response")
	}
	xVerify := c.Get("X-VERIFY")

	redirectURL, err := h.payments.HandleReturn(c.Context(), token, base64Resp, xVerify)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).SendString("Payment not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("Unable to process return")
	}
	return c.Redirect(redirectURL, fiber.StatusTemporaryRedirect)
}

type PhonePeWebhookHandler struct {
	payments *services.PaymentService
}

func NewPhonePeWebhookHandler(payments *services.PaymentService) *PhonePeWebhookHandler {
	return &PhonePeWebhookHandler{payments: payments}
}

func (h *PhonePeWebhookHandler) Handle(c *fiber.Ctx) error {
	var body struct {
		Response string `json:"response"`
	}
	if err := c.BodyParser(&body); err != nil || body.Response == "" {
		// PhonePe may send raw base64 in some modes
		body.Response = string(c.Body())
	}

	xVerify := c.Get("X-VERIFY")
	if err := h.payments.HandlePhonePeWebhook(c.Context(), body.Response, xVerify); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}
