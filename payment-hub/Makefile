.PHONY: help run build tidy test docker-up docker-down migrate-up migrate-down migrate-docker sqlc

help:
	@echo "Payment Hub — available commands:"
	@echo "  make docker-up     Start MySQL + Redis (local dev)"
	@echo "  make docker-down   Stop Docker services"
	@echo "  make migrate-up    Run DB migrations (requires migrate CLI)"
	@echo "  make migrate-docker Run migrations via Docker"
	@echo "  make run           Start Go server"
	@echo "  make build         Build binary"
	@echo "  make tidy          go mod tidy"
	@echo "  make test          Run tests"

docker-up:
	docker compose up -d

docker-down:
	docker compose down

# Requires: brew install golang-migrate
# Or: go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
migrate-up:
	migrate -path migrations -database "mysql://payment_hub:payment_hub@tcp(127.0.0.1:3306)/payment_hub?multiStatements=true" up

migrate-down:
	migrate -path migrations -database "mysql://payment_hub:payment_hub@tcp(127.0.0.1:3306)/payment_hub?multiStatements=true" down 1

migrate-docker:
	docker run --rm -v "$(PWD)/migrations:/migrations" --network host migrate/migrate \
		-path=/migrations -database "mysql://payment_hub:payment_hub@tcp(127.0.0.1:3306)/payment_hub?multiStatements=true" up

run:
	go run ./cmd/server

build:
	go build -o bin/payment-hub ./cmd/server

tidy:
	go mod tidy

test:
	go test ./...

sqlc:
	sqlc generate
