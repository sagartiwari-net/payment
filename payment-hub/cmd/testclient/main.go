package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/buyahref/payment-hub/internal/security"
)

func main() {
	baseURL := env("BASE_URL", "https://buyahref.com/payment")
	apiKey := env("API_KEY", "mk_semrushtoolz_001")
	apiSecret := env("API_SECRET", "sk_semrushtoolz_secret_change_me_in_production")

	orderID := fmt.Sprintf("TEST-%d", time.Now().Unix())
	body := map[string]interface{}{
		"order_id": orderID,
		"amount":   1.00,
		"currency": "INR",
		"customer": map[string]string{
			"email": "test@example.com",
			"name":  "Test User",
		},
		"product": map[string]string{
			"name": "Test Product",
		},
		"return_url":  "https://semrushtoolz.com/amember/payment/buyahref/return",
		"webhook_url": "https://semrushtoolz.com/amember/payment/buyahref/webhook",
	}

	bodyBytes, _ := json.Marshal(body)
	createPath := "/api/v1/orders/create"

	fmt.Println("=== Create Order ===")
	createResp, err := signedRequest(http.MethodPost, baseURL+createPath, createPath, bodyBytes, apiKey, apiSecret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(createResp)

	verifyPath := "/api/v1/orders/" + orderID + "/verify"
	fmt.Println("\n=== Verify Order ===")
	verifyResp, err := signedRequest(http.MethodGet, baseURL+verifyPath, verifyPath, nil, apiKey, apiSecret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "verify failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(verifyResp)
}

func signedRequest(method, url, path string, body []byte, apiKey, apiSecret string) (string, error) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sig := security.Sign(apiSecret, ts, method, path, string(body))

	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Merchant-Key", apiKey)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Signature", sig)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("HTTP %d %s", resp.StatusCode, string(data)), nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
