package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func Sign(secret, timestamp, method, path, body string) string {
	message := fmt.Sprintf("%s|%s|%s|%s", timestamp, method, path, body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func Verify(secret, timestamp, method, path, body, signature string) bool {
	expected := Sign(secret, timestamp, method, path, body)
	return hmac.Equal([]byte(expected), []byte(signature))
}
