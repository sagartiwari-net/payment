#!/bin/bash
# Test Phase 2 APIs — uses Go (no openssl/awk required)
set -e

cd "$(dirname "$0")/.."

export BASE_URL="${BASE_URL:-https://buyahref.com/payment}"
export API_KEY="${API_KEY:-mk_semrushtoolz_001}"
export API_SECRET="${API_SECRET:-sk_semrushtoolz_secret_change_me_in_production}"

go run ./cmd/testclient
