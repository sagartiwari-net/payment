package phonepe

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestChecksumFormat(t *testing.T) {
	cfg := Config{SaltKey: "test-key", SaltIndex: "1"}
	c := NewClient(cfg)

	base64Payload := "eyJ0ZXN0IjoidHJ1ZSJ9"
	checksum := c.checksum(base64Payload, payPathPrefix)

	parts := splitChecksum(checksum)
	if len(parts) != 2 || parts[1] != "1" {
		t.Fatalf("unexpected checksum format: %s", checksum)
	}
	if len(parts[0]) != 64 {
		t.Fatalf("expected sha256 hex length 64, got %d", len(parts[0]))
	}
}

func TestCallbackChecksumDeterministic(t *testing.T) {
	cfg := Config{SaltKey: "salt", SaltIndex: "1"}
	c := NewClient(cfg)
	a := c.callbackChecksum("abc123")
	b := c.callbackChecksum("abc123")
	if a != b {
		t.Fatal("checksum should be deterministic")
	}
}

func splitChecksum(s string) []string {
	for i := 0; i < len(s); i++ {
		if i+3 <= len(s) && s[i:i+3] == "###" {
			return []string{s[:i], s[i+3:]}
		}
	}
	return nil
}

func TestIsPaymentSuccess(t *testing.T) {
	p := &CallbackPayload{Code: "PAYMENT_SUCCESS"}
	if !IsPaymentSuccess(p) {
		t.Fatal("expected success")
	}
}

func TestBaseURL(t *testing.T) {
	if BaseURL("PRODUCTION") != "https://api.phonepe.com/apis/hermes" {
		t.Fatal("production url mismatch")
	}
	if BaseURL("UAT") != "https://api-preprod.phonepe.com/apis/hermes" {
		t.Fatal("uat url mismatch")
	}
}

func BenchmarkChecksum(b *testing.B) {
	cfg := Config{SaltKey: "key", SaltIndex: "1"}
	c := NewClient(cfg)
	for i := 0; i < b.N; i++ {
		_ = c.checksum("payload", payPathPrefix)
	}
}

func init() {
	_ = sha256.Sum256
	_ = hex.EncodeToString
}
