package security

import "testing"

func TestSignAndVerify(t *testing.T) {
	secret := "test_secret_key"
	timestamp := "1717654321"
	method := "POST"
	path := "/api/v1/orders/create"
	body := `{"order_id":"INV-1","amount":999}`

	sig := Sign(secret, timestamp, method, path, body)
	if !Verify(secret, timestamp, method, path, body, sig) {
		t.Fatal("expected valid signature")
	}
}

func TestVerifyRejectsTamperedBody(t *testing.T) {
	secret := "test_secret_key"
	timestamp := "1717654321"
	method := "POST"
	path := "/api/v1/orders/create"
	body := `{"order_id":"INV-1","amount":999}`

	sig := Sign(secret, timestamp, method, path, body)
	if Verify(secret, timestamp, method, path, `{"order_id":"INV-1","amount":1}`, sig) {
		t.Fatal("expected invalid signature for tampered body")
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	timestamp := "1717654321"
	method := "GET"
	path := "/api/v1/orders/INV-1/verify"
	body := ""

	sig := Sign("secret_a", timestamp, method, path, body)
	if Verify("secret_b", timestamp, method, path, body, sig) {
		t.Fatal("expected invalid signature for wrong secret")
	}
}
