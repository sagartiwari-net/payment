#!/bin/bash
# Run on VPS after git clone (no root required)
set -e

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_DIR"

echo "=== Payment Hub VPS Setup ==="

if [ ! -f .env ]; then
  echo "Creating .env from template..."
  cp env.production.template .env
  echo ""
  echo "IMPORTANT: Edit .env and set DB_PASSWORD and PHONEPE_SALT_KEY"
  echo "  nano .env"
  echo ""
  exit 1
fi

if ! command -v go &> /dev/null; then
  echo "Go is not installed. Install Go 1.22+ first:"
  echo "  https://go.dev/dl/"
  exit 1
fi

echo "Installing dependencies..."
go mod tidy

echo "Building..."
mkdir -p bin
go build -o bin/payment-hub ./cmd/server

echo ""
echo "Build OK: bin/payment-hub"
echo ""
echo "Start server:"
echo "  cd $REPO_DIR && ./bin/payment-hub"
echo ""
echo "Or run in background:"
echo "  nohup ./bin/payment-hub > payment-hub.log 2>&1 &"
echo ""
echo "Test health:"
echo "  curl http://127.0.0.1:8090/health"
