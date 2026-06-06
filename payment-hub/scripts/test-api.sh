#!/bin/bash
# Test Phase 2 APIs locally or on VPS
set -e

BASE_URL="${BASE_URL:-https://buyahref.com/payment}"
API_KEY="${API_KEY:-mk_semrushtoolz_001}"
API_SECRET="${API_SECRET:-sk_semrushtoolz_secret_change_me_in_production}"

TIMESTAMP=$(date +%s)
METHOD="POST"
PATH="/api/v1/orders/create"
BODY='{"order_id":"TEST-'$TIMESTAMP'","amount":1.00,"currency":"INR","customer":{"email":"test@example.com","name":"Test User"},"product":{"name":"Test Product"},"return_url":"https://semrushtoolz.com/amember/payment/buyahref/return","webhook_url":"https://semrushtoolz.com/amember/payment/buyahref/webhook"}'

MESSAGE="${TIMESTAMP}|${METHOD}|${PATH}|${BODY}"
SIGNATURE=$(printf '%s' "$MESSAGE" | openssl dgst -sha256 -hmac "$API_SECRET" | awk '{print $2}')

echo "=== Create Order ==="
CREATE_RESP=$(curl -s -X POST "${BASE_URL}${PATH}" \
  -H "Content-Type: application/json" \
  -H "X-Merchant-Key: ${API_KEY}" \
  -H "X-Timestamp: ${TIMESTAMP}" \
  -H "X-Signature: ${SIGNATURE}" \
  -d "$BODY")
echo "$CREATE_RESP"

ORDER_ID="TEST-${TIMESTAMP}"
VERIFY_PATH="/api/v1/orders/${ORDER_ID}/verify"
VERIFY_TS=$(date +%s)
VERIFY_MSG="${VERIFY_TS}|GET|${VERIFY_PATH}|"
VERIFY_SIG=$(printf '%s' "$VERIFY_MSG" | openssl dgst -sha256 -hmac "$API_SECRET" | awk '{print $2}')

echo ""
echo "=== Verify Order ==="
curl -s "${BASE_URL}${VERIFY_PATH}" \
  -H "X-Merchant-Key: ${API_KEY}" \
  -H "X-Timestamp: ${VERIFY_TS}" \
  -H "X-Signature: ${VERIFY_SIG}"
echo ""
