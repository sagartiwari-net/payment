package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/buyahref/payment-hub/internal/models"
	"github.com/buyahref/payment-hub/internal/repository"
	"github.com/buyahref/payment-hub/internal/security"
)

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrMerchantSuspended = errors.New("merchant suspended")
	ErrDuplicateOrder    = errors.New("duplicate order")
)

type OrderService struct {
	orders   *repository.OrderRepository
	appURL   string
	expiry   time.Duration
}

func NewOrderService(orders *repository.OrderRepository, appURL string, expiryMinutes int) *OrderService {
	if expiryMinutes <= 0 {
		expiryMinutes = 30
	}
	return &OrderService{
		orders: orders,
		appURL: strings.TrimRight(appURL, "/"),
		expiry: time.Duration(expiryMinutes) * time.Minute,
	}
}

func (s *OrderService) Create(ctx context.Context, merchant *models.Merchant, in models.CreateOrderInput) (*models.CreateOrderResult, error) {
	if merchant.Status != models.MerchantStatusActive {
		return nil, ErrMerchantSuspended
	}
	if err := validateCreateInput(&in); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	hubOrderID, err := s.orders.AllocateHubOrderID(ctx)
	if err != nil {
		return nil, err
	}

	token, err := security.NewPaymentToken()
	if err != nil {
		return nil, err
	}

	webhookURL := in.WebhookURL
	if webhookURL == "" {
		webhookURL = merchant.WebhookURL
	}

	expiresAt := time.Now().UTC().Add(s.expiry)

	order := &models.Order{
		ID:                 security.NewID(),
		HubOrderID:         hubOrderID,
		MerchantID:         merchant.ID,
		MerchantOrderID:    in.OrderID,
		PaymentToken:       token,
		Amount:             in.Amount,
		Currency:           in.Currency,
		Status:             models.OrderStatusPending,
		CustomerEmail:      in.Customer.Email,
		CustomerName:       in.Customer.Name,
		CustomerPhone:      in.Customer.Phone,
		ProductName:        in.Product.Name,
		ProductDescription: in.Product.Description,
		ReturnURL:          in.ReturnURL,
		WebhookURL:         webhookURL,
		ExpiresAt:          expiresAt,
	}

	if err := s.orders.Create(ctx, order); err != nil {
		if errors.Is(err, repository.ErrDuplicateOrder) {
			return nil, ErrDuplicateOrder
		}
		return nil, err
	}

	return &models.CreateOrderResult{
		HubOrderID: hubOrderID,
		PaymentURL: fmt.Sprintf("%s/pay/%s", s.appURL, token),
		ExpiresAt:  expiresAt,
	}, nil
}

func (s *OrderService) Verify(ctx context.Context, merchant *models.Merchant, merchantOrderID string) (*models.VerifyOrderResult, error) {
	order, err := s.orders.GetByMerchantOrderID(ctx, merchant.ID, merchantOrderID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &models.VerifyOrderResult{
		HubOrderID:   order.HubOrderID,
		OrderID:      order.MerchantOrderID,
		Status:       order.Status,
		Amount:       order.Amount,
		Currency:     order.Currency,
		PhonePeTxnID: order.PhonePeTxnID,
		PaidAt:       order.PaidAt,
	}, nil
}

func validateCreateInput(in *models.CreateOrderInput) error {
	if strings.TrimSpace(in.OrderID) == "" {
		return errors.New("order_id is required")
	}
	if len(in.OrderID) > 100 {
		return errors.New("order_id too long")
	}
	if in.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if in.Currency == "" {
		in.Currency = "INR"
	}
	if in.Currency != "INR" {
		return errors.New("only INR currency supported")
	}
	if strings.TrimSpace(in.ReturnURL) == "" {
		return errors.New("return_url is required")
	}
	if _, err := url.ParseRequestURI(in.ReturnURL); err != nil {
		return errors.New("return_url is invalid")
	}
	if in.WebhookURL != "" {
		if _, err := url.ParseRequestURI(in.WebhookURL); err != nil {
			return errors.New("webhook_url is invalid")
		}
	}
	return nil
}
