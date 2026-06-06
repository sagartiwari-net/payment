package phonepe

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	payPathPrefix    = "/pg/v1/pay"
	statusPathPrefix = "/pg/v1/status"
)

type Config struct {
	MerchantID string
	SaltKey    string
	SaltIndex  string
	BaseURL    string
}

type Client struct {
	cfg    Config
	client *http.Client
}

func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.phonepe.com/apis/hermes"
	}
	return &Client{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type PayRequest struct {
	MerchantTransactionID string
	MerchantUserID        string
	AmountPaise           int64
	RedirectURL           string
	CallbackURL           string
}

type PayResponse struct {
	Success bool
	Code    string
	Message string
	PayURL  string
	Raw     json.RawMessage
}

type StatusResponse struct {
	Success      bool
	Code         string
	State        string
	ResponseCode string
	TxnID        string
	AmountPaise  int64
	Raw          json.RawMessage
}

type CallbackPayload struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		MerchantID            string `json:"merchantId"`
		MerchantTransactionID string `json:"merchantTransactionId"`
		TransactionID         string `json:"transactionId"`
		Amount                int64  `json:"amount"`
		State                 string `json:"state"`
		ResponseCode          string `json:"responseCode"`
	} `json:"data"`
}

func (c *Client) Pay(ctx context.Context, req PayRequest) (*PayResponse, error) {
	payload := map[string]interface{}{
		"merchantId":            c.cfg.MerchantID,
		"merchantTransactionId": req.MerchantTransactionID,
		"merchantUserId":        req.MerchantUserID,
		"amount":                req.AmountPaise,
		"redirectUrl":           req.RedirectURL,
		"redirectMode":          "POST",
		"callbackUrl":           req.CallbackURL,
		"paymentInstrument": map[string]string{
			"type": "PAY_PAGE",
		},
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	base64Payload := base64.StdEncoding.EncodeToString(payloadJSON)
	checksum := c.checksum(base64Payload, payPathPrefix)

	reqBody, _ := json.Marshal(map[string]string{"request": base64Payload})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+payPathPrefix, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-VERIFY", checksum)
	httpReq.Header.Set("X-MERCHANT-ID", c.cfg.MerchantID)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("phonepe pay request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("phonepe pay http %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			InstrumentResponse struct {
				RedirectInfo struct {
					URL    string `json:"url"`
					Method string `json:"method"`
				} `json:"redirectInfo"`
			} `json:"instrumentResponse"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse pay response: %w", err)
	}

	payURL := parsed.Data.InstrumentResponse.RedirectInfo.URL
	if payURL == "" {
		return nil, fmt.Errorf("phonepe pay: missing redirect url (%s)", string(body))
	}

	return &PayResponse{
		Success: parsed.Success,
		Code:    parsed.Code,
		Message: parsed.Message,
		PayURL:  payURL,
		Raw:     json.RawMessage(body),
	}, nil
}

func (c *Client) Status(ctx context.Context, merchantTransactionID string) (*StatusResponse, error) {
	path := fmt.Sprintf("%s/%s/%s", statusPathPrefix, c.cfg.MerchantID, merchantTransactionID)
	checksum := c.statusChecksum(path)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-VERIFY", checksum)
	httpReq.Header.Set("X-MERCHANT-ID", c.cfg.MerchantID)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("phonepe status request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			TransactionID string `json:"transactionId"`
			Amount        int64  `json:"amount"`
			State         string `json:"state"`
			ResponseCode  string `json:"responseCode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse status response: %w", err)
	}

	return &StatusResponse{
		Success:      parsed.Success,
		Code:         parsed.Code,
		State:        parsed.Data.State,
		ResponseCode: parsed.Data.ResponseCode,
		TxnID:        parsed.Data.TransactionID,
		AmountPaise:  parsed.Data.Amount,
		Raw:          json.RawMessage(body),
	}, nil
}

func (c *Client) VerifyCallbackSignature(base64Response, xVerify string) bool {
	expected := c.callbackChecksum(base64Response)
	return hmacEqual(expected, xVerify)
}

func (c *Client) DecodeCallback(base64Response string) (*CallbackPayload, error) {
	decoded, err := base64.StdEncoding.DecodeString(base64Response)
	if err != nil {
		return nil, fmt.Errorf("decode callback: %w", err)
	}
	var payload CallbackPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return nil, fmt.Errorf("parse callback: %w", err)
	}
	return &payload, nil
}

func (c *Client) checksum(base64Payload, path string) string {
	data := base64Payload + path + c.cfg.SaltKey
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:]) + "###" + c.cfg.SaltIndex
}

func (c *Client) statusChecksum(path string) string {
	data := path + c.cfg.SaltKey
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:]) + "###" + c.cfg.SaltIndex
}

func (c *Client) callbackChecksum(base64Response string) string {
	data := base64Response + c.cfg.SaltKey
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:]) + "###" + c.cfg.SaltIndex
}

func hmacEqual(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func IsPaymentSuccess(payload *CallbackPayload) bool {
	if payload == nil {
		return false
	}
	if payload.Code == "PAYMENT_SUCCESS" || payload.Data.ResponseCode == "SUCCESS" {
		return true
	}
	return payload.Success && payload.Data.State == "COMPLETED"
}

func IsStatusSuccess(status *StatusResponse) bool {
	if status == nil {
		return false
	}
	if status.Code == "PAYMENT_SUCCESS" || status.ResponseCode == "SUCCESS" {
		return true
	}
	return status.Success && status.State == "COMPLETED"
}

func BaseURL(env string) string {
	switch strings.ToUpper(env) {
	case "UAT", "SANDBOX", "TEST":
		return "https://api-preprod.phonepe.com/apis/hermes"
	default:
		return "https://api.phonepe.com/apis/hermes"
	}
}
