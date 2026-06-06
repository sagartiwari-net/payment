package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/buyahref/payment-hub/internal/models"
	"github.com/buyahref/payment-hub/internal/order"
	"github.com/buyahref/payment-hub/internal/phonepe"
	"github.com/buyahref/payment-hub/internal/repository"
	"github.com/buyahref/payment-hub/internal/security"
	"go.uber.org/zap"
)

var (
	ErrOrderExpired   = errors.New("order expired")
	ErrOrderNotPayable = errors.New("order not payable")
)

type PaymentService struct {
	orders    *repository.OrderRepository
	merchants *repository.MerchantRepository
	phonepe   *phonepe.Client
	appURL    string
	log       *zap.Logger
}

func NewPaymentService(
	orders *repository.OrderRepository,
	merchants *repository.MerchantRepository,
	pp *phonepe.Client,
	appURL string,
	log *zap.Logger,
) *PaymentService {
	return &PaymentService{
		orders:    orders,
		merchants: merchants,
		phonepe:   pp,
		appURL:    strings.TrimRight(appURL, "/"),
		log:       log,
	}
}

type CheckoutView struct {
	Token         string
	ProductName   string
	Amount        string
	Merchant      string
	Status        string
	Expired       bool
	Payable       bool
	PayActionURL  string
}

func (s *PaymentService) CheckoutURL(token string) string {
	return fmt.Sprintf("%s/pay/%s", s.appURL, token)
}

func (s *PaymentService) GetCheckout(ctx context.Context, token string) (*CheckoutView, *models.Order, error) {
	o, err := s.orders.GetByPaymentToken(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	merchant, err := s.merchants.GetByID(ctx, o.MerchantID)
	if err != nil {
		return nil, nil, err
	}

	if o.Status == models.OrderStatusPending && time.Now().UTC().After(o.ExpiresAt) {
		_ = s.orders.TransitionStatus(ctx, o.ID, models.OrderStatusPending, models.OrderStatusExpired)
		o.Status = models.OrderStatusExpired
	}

	view := &CheckoutView{
		Token:        token,
		ProductName:  fallback(o.ProductName, "Payment"),
		Amount:       fmt.Sprintf("%.2f", o.Amount),
		Merchant:     merchant.Domain,
		Status:       o.Status,
		Expired:      o.Status == models.OrderStatusExpired,
		Payable:      o.Status == models.OrderStatusPending,
		PayActionURL: fmt.Sprintf("%s/pay/%s/pay", s.appURL, token),
	}
	return view, o, nil
}

func (s *PaymentService) InitiatePayment(ctx context.Context, token string) (string, error) {
	o, err := s.orders.GetByPaymentToken(ctx, token)
	if err != nil {
		return "", err
	}

	if o.Status == models.OrderStatusPending && time.Now().UTC().After(o.ExpiresAt) {
		_ = s.orders.TransitionStatus(ctx, o.ID, models.OrderStatusPending, models.OrderStatusExpired)
		return "", ErrOrderExpired
	}
	if o.Status != models.OrderStatusPending {
		return "", ErrOrderNotPayable
	}

	userID := fallback(o.CustomerEmail, o.CustomerPhone)
	if userID == "" {
		userID = "guest_" + o.HubOrderID
	}

	amountPaise := int64(o.Amount * 100)
	resp, err := s.phonepe.Pay(ctx, phonepe.PayRequest{
		MerchantTransactionID: o.HubOrderID,
		MerchantUserID:        userID,
		AmountPaise:           amountPaise,
		RedirectURL:           fmt.Sprintf("%s/pay/%s/return", s.appURL, token),
		CallbackURL:           fmt.Sprintf("%s/webhooks/phonepe", s.appURL),
	})
	if err != nil {
		return "", err
	}

	_ = s.orders.SavePhonePeResponse(ctx, o.ID, resp.Raw)
	if err := s.orders.TransitionStatus(ctx, o.ID, models.OrderStatusPending, models.OrderStatusProcessing); err != nil {
		// already processing — allow retry
		if o.Status != models.OrderStatusProcessing {
			return "", err
		}
	}

	return resp.PayURL, nil
}

func (s *PaymentService) HandlePhonePeWebhook(ctx context.Context, base64Response, xVerify string) error {
	if !s.phonepe.VerifyCallbackSignature(base64Response, xVerify) {
		return errors.New("invalid phonepe signature")
	}

	payload, err := s.phonepe.DecodeCallback(base64Response)
	if err != nil {
		return err
	}

	return s.applyPhonePeResult(ctx, payload.Data.MerchantTransactionID, phonepe.IsPaymentSuccess(payload), payload.Data.TransactionID, payload.Data.Amount, mustJSON(payload))
}

func (s *PaymentService) HandleReturn(ctx context.Context, token, base64Response, xVerify string) (string, error) {
	o, err := s.orders.GetByPaymentToken(ctx, token)
	if err != nil {
		return "", err
	}

	if base64Response != "" {
		if s.phonepe.VerifyCallbackSignature(base64Response, xVerify) {
			payload, err := s.phonepe.DecodeCallback(base64Response)
			if err == nil {
				_ = s.applyPhonePeResult(ctx, payload.Data.MerchantTransactionID, phonepe.IsPaymentSuccess(payload), payload.Data.TransactionID, payload.Data.Amount, mustJSON(payload))
			}
		}
	}

	// refresh order
	o, err = s.orders.GetByPaymentToken(ctx, token)
	if err != nil {
		return "", err
	}

	if o.Status == models.OrderStatusProcessing {
		status, err := s.phonepe.Status(ctx, o.HubOrderID)
		if err == nil {
			success := phonepe.IsStatusSuccess(status)
			_ = s.applyPhonePeResult(ctx, o.HubOrderID, success, status.TxnID, status.AmountPaise, status.Raw)
			o, _ = s.orders.GetByPaymentToken(ctx, token)
		}
	}

	return s.buildReturnURL(o), nil
}

func (s *PaymentService) applyPhonePeResult(ctx context.Context, hubOrderID string, success bool, txnID string, amountPaise int64, raw []byte) error {
	o, err := s.orders.GetByHubOrderID(ctx, hubOrderID)
	if err != nil {
		return err
	}

	if order.IsFinalStatus(o.Status) {
		return nil
	}

	if amountPaise > 0 {
		expected := int64(o.Amount * 100)
		if amountPaise != expected {
			s.log.Warn("phonepe amount mismatch",
				zap.String("hub_order_id", hubOrderID),
				zap.Int64("expected_paise", expected),
				zap.Int64("got_paise", amountPaise),
			)
			return errors.New("amount mismatch")
		}
	}

	if success {
		if err := s.orders.MarkSuccess(ctx, o.ID, txnID, raw); err != nil {
			return err
		}
		updated, err := s.orders.GetByHubOrderID(ctx, hubOrderID)
		if err == nil {
			go s.notifyMerchant(updated, "payment.success")
		}
		return nil
	}

	if err := s.orders.MarkFailed(ctx, o.ID, raw); err != nil {
		return err
	}
	updated, err := s.orders.GetByHubOrderID(ctx, hubOrderID)
	if err == nil {
		go s.notifyMerchant(updated, "payment.failed")
	}
	return nil
}

func (s *PaymentService) buildReturnURL(o *models.Order) string {
	u, err := url.Parse(o.ReturnURL)
	if err != nil {
		return o.ReturnURL
	}
	q := u.Query()
	q.Set("order_id", o.MerchantOrderID)
	q.Set("status", o.Status)
	q.Set("hub_order_id", o.HubOrderID)
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *PaymentService) notifyMerchant(o *models.Order, event string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	merchant, err := s.merchants.GetByID(ctx, o.MerchantID)
	if err != nil {
		s.log.Error("notify merchant: load merchant", zap.Error(err))
		return
	}

	webhookURL := o.WebhookURL
	if webhookURL == "" {
		webhookURL = merchant.WebhookURL
	}
	if webhookURL == "" {
		return
	}

	payload := map[string]interface{}{
		"event":        event,
		"hub_order_id": o.HubOrderID,
		"order_id":     o.MerchantOrderID,
		"amount":       o.Amount,
		"currency":     o.Currency,
		"status":       o.Status,
	}
	if o.PhonePeTxnID != "" {
		payload["phonepe_txn_id"] = o.PhonePeTxnID
	}
	if o.PaidAt != nil {
		payload["paid_at"] = o.PaidAt.UTC().Format(time.RFC3339)
	}

	bodyBytes, _ := json.Marshal(payload)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	whURL, _ := url.Parse(webhookURL)
	path := "/"
	if whURL != nil && whURL.Path != "" {
		path = whURL.Path
	}
	sig := security.Sign(merchant.APISecret, ts, http.MethodPost, path, string(bodyBytes))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature", sig)
	req.Header.Set("X-Hub-Timestamp", ts)
	req.Header.Set("X-Hub-Event", event)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.log.Error("notify merchant failed", zap.Error(err), zap.String("url", webhookURL))
		return
	}
	defer resp.Body.Close()
	s.log.Info("merchant webhook sent", zap.Int("status", resp.StatusCode), zap.String("event", event))
}

func fallback(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

var checkoutTemplate = template.Must(template.New("checkout").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Secure Payment</title>
  <style>
    body { font-family: system-ui, sans-serif; background:#f4f6fb; display:flex; min-height:100vh; align-items:center; justify-content:center; margin:0; }
    .card { background:#fff; padding:32px; border-radius:12px; box-shadow:0 8px 30px rgba(0,0,0,.08); max-width:420px; width:100%; }
    h1 { font-size:22px; margin:0 0 8px; }
    .muted { color:#666; font-size:14px; margin-bottom:24px; }
    .row { display:flex; justify-content:space-between; margin:12px 0; font-size:15px; }
    .amount { font-size:28px; font-weight:700; color:#5f259f; }
    button { width:100%; margin-top:24px; padding:14px; border:0; border-radius:8px; background:#5f259f; color:#fff; font-size:16px; font-weight:600; cursor:pointer; }
    button:disabled { background:#ccc; cursor:not-allowed; }
    .badge { display:inline-block; padding:4px 10px; border-radius:999px; background:#eee; font-size:12px; text-transform:uppercase; }
  </style>
</head>
<body>
  <div class="card">
    <h1>{{.ProductName}}</h1>
    <div class="muted">via {{.Merchant}} · secured by buyahref.com</div>
    <div class="row"><span>Amount</span><span class="amount">₹{{.Amount}}</span></div>
    <div class="row"><span>Status</span><span class="badge">{{.Status}}</span></div>
    {{if .Payable}}
    <form method="POST" action="{{.PayActionURL}}">
      <button type="submit">Pay with PhonePe</button>
    </form>
    {{else if .Expired}}
    <p class="muted">This payment link has expired. Please create a new order.</p>
    {{else}}
    <p class="muted">This payment is no longer available.</p>
    {{end}}
  </div>
</body>
</html>`))

func RenderCheckoutHTML(view *CheckoutView) (string, error) {
	var buf bytes.Buffer
	if err := checkoutTemplate.Execute(&buf, view); err != nil {
		return "", err
	}
	return buf.String(), nil
}
