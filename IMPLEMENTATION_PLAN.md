# Buyahref Payment Hub — Complete Implementation Plan

> **Goal:** Standalone Go backend on `buyahref.com` that accepts PhonePe payments on behalf of multiple merchant websites (semrushtoolz.com + 3-4 others), with a central admin dashboard for all transactions.

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Architecture](#2-architecture)
3. [Tech Stack](#3-tech-stack)
4. [Repository Structure](#4-repository-structure)
5. [Database Schema](#5-database-schema)
6. [API Specification](#6-api-specification)
7. [Payment Flow (Step-by-Step)](#7-payment-flow-step-by-step)
8. [Security Specification](#8-security-specification)
9. [PhonePe Integration](#9-phonepe-integration)
10. [Admin Dashboard Plan](#10-admin-dashboard-plan)
11. [Merchant Integration Guides](#11-merchant-integration-guides)
12. [Implementation Phases](#12-implementation-phases)
13. [Testing Checklist](#13-testing-checklist)
14. [Deployment Guide](#14-deployment-guide)
15. [Environment Variables](#15-environment-variables)
16. [Future Roadmap](#16-future-roadmap)

---

## 1. Project Overview

### Problem
- PhonePe approves **only 1 domain per account** (`buyahref.com` exactly — no subdomains).
- 3-4 other websites need payment acceptance but cannot get their own PhonePe gateway.
- All merchant sites use **aMember Pro (PHP)** now; custom **React/Next.js** dashboards planned next month.

### Solution
Build a **standalone Payment Hub** on `buyahref.com`:
- All PhonePe interactions happen only on `buyahref.com`.
- Merchant sites call Hub API → redirect user to Hub checkout → user pays → returns to merchant site.
- Central dashboard tracks all payments across all domains.

### Key Constraints
| Constraint | Detail |
|------------|--------|
| PhonePe domain | Payment page MUST be served from `buyahref.com` (exact domain, no subdomain) |
| Concurrency | Multiple users from same site can pay simultaneously — fully isolated |
| Integration | Must work with PHP (aMember) now AND React/Next.js later |
| Security | No fake payments — webhook + verify API mandatory before order fulfillment |

### Projects (2 Repositories)

```
payment-hub/          → Go backend (API + PhonePe + webhooks)
payment-hub-admin/    → Next.js admin dashboard
payment-hub-sdk-php/  → Thin PHP SDK for aMember plugin
```

---

## 2. Architecture

### High-Level Diagram

```
┌──────────────────────────────────────────────────────────────────┐
│                     buyahref.com                                  │
│  ┌─────────────────────┐    ┌─────────────────────────────────┐  │
│  │  Go Backend (API)   │    │  Next.js Admin Dashboard        │  │
│  │  :8080 (internal)   │◄───│  /admin                         │  │
│  │                     │    │  Stats, transactions, merchants │  │
│  │  • REST API v1      │    └─────────────────────────────────┘  │
│  │  • PhonePe client   │                                          │
│  │  • Webhook handler  │    ┌─────────────────────────────────┐  │
│  │  • Checkout page    │    │  Checkout Page (Go template)    │  │
│  │  • Job queue worker │    │  /pay/{token}                   │  │
│  └──────────┬──────────┘    └─────────────────────────────────┘  │
│             │                                                       │
│  ┌──────────▼──────────┐    ┌─────────────────────────────────┐  │
│  │  MySQL (VPS)        │    │  Redis                          │  │
│  │  orders, merchants  │    │  rate limits, job queue         │  │
│  │  phpMyAdmin admin   │    │                                 │  │
│  └─────────────────────┘    └─────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────────┐
              │               │                   │
              ▼               ▼                   ▼
     ┌─────────────┐  ┌─────────────┐    ┌─────────────┐
     │semrushtoolz │  │ website2    │    │ website3    │
     │ aMember PHP │  │ aMember PHP │    │ Next.js     │
     │ (now)       │  │ (now)       │    │ (future)    │
     └─────────────┘  └─────────────┘    └─────────────┘
```

### Request Isolation Model

```
Merchant Isolation     →  Each site has unique API key, can only see own data
Order Isolation        →  Each payment has unique token, independent state
Request Isolation      →  Stateless API, row-level DB locks per order
```

### Order State Machine

```
                    create-order
                         │
                         ▼
                    ┌─────────┐
         ┌─────────│ PENDING │─────────┐
         │         └────┬────┘         │
         │              │              │ (30 min timeout)
         │    user pays │              ▼
         │              ▼         ┌─────────┐
         │       ┌────────────┐   │ EXPIRED │
         │       │ PROCESSING │   └─────────┘
         │       └─────┬──────┘
         │             │
         │   ┌─────────┼─────────┐
         │   ▼         ▼         ▼
         │ SUCCESS   FAILED   (final states — no transitions out)
         │
         └── refund from SUCCESS → REFUNDED
```

**Rules:**
- Only `pending` → `processing` → `success|failed` allowed.
- Duplicate webhook for same order = idempotent (ignore if already final).
- `expired` = auto-set by cron after 30 minutes in `pending`.

---

## 3. Tech Stack

### Payment Hub Backend
| Component | Technology | Version |
|-----------|-----------|---------|
| Language | Go | 1.22+ |
| HTTP Framework | Fiber | v2 |
| Database | MySQL | 8.0+ (existing VPS + phpMyAdmin) |
| Query Layer | sqlc | latest |
| Migrations | golang-migrate | latest |
| Cache / Queue | Redis + Asynq | latest |
| Config | Viper | latest |
| Logging | Zap | latest |
| Validation | go-playground/validator | latest |

### Admin Dashboard
| Component | Technology |
|-----------|-----------|
| Framework | Next.js 14 (App Router) |
| UI | Tailwind CSS + shadcn/ui |
| Charts | Recharts |
| Auth | NextAuth.js (credentials → Hub admin API) |
| HTTP Client | fetch / axios |

### Merchant Integrations
| Site Type | Integration |
|-----------|-------------|
| aMember Pro (now) | Thin PHP plugin using `payment-hub-sdk-php` |
| Next.js (future) | TypeScript API client (same REST endpoints) |
| Any other | REST API + HMAC signing (documented) |

### Infrastructure
| Component | Technology |
|-----------|-----------|
| Reverse Proxy | Nginx |
| Containerization | Docker + Docker Compose |
| SSL | Let's Encrypt (Certbot) |
| Server | VPS (buyahref.com) |

---

## 4. Repository Structure

### `payment-hub/` (Go Backend)

```
payment-hub/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go               # Env loading via Viper
│   ├── database/
│   │   ├── db.go                   # MySQL connection pool
│   │   └── queries/                # sqlc generated queries
│   ├── models/
│   │   └── models.go               # Domain structs
│   ├── api/
│   │   ├── routes.go               # Route registration
│   │   ├── middleware/
│   │   │   ├── auth.go             # API key + HMAC verification
│   │   │   ├── ratelimit.go        # Redis rate limiter
│   │   │   ├── cors.go             # CORS for allowed origins
│   │   │   └── logger.go           # Request logging
│   │   └── handlers/
│   │       ├── order_create.go     # POST /api/v1/orders/create
│   │       ├── order_verify.go     # GET  /api/v1/orders/:id/verify
│   │       ├── order_refund.go     # POST /api/v1/orders/:id/refund
│   │       ├── checkout.go         # GET  /pay/:token (HTML page)
│   │       ├── phonepe_webhook.go  # POST /webhooks/phonepe
│   │       └── admin/              # Admin-only endpoints
│   │           ├── auth.go
│   │           ├── stats.go
│   │           ├── merchants.go
│   │           ├── transactions.go
│   │           └── refunds.go
│   ├── services/
│   │   ├── order_service.go        # Business logic + state machine
│   │   ├── merchant_service.go
│   │   ├── webhook_service.go      # Outbound webhooks to merchants
│   │   └── refund_service.go
│   ├── phonepe/
│   │   ├── client.go               # PhonePe API client
│   │   ├── webhook_verify.go       # X-VERIFY header validation
│   │   └── types.go
│   ├── security/
│   │   ├── hmac.go                 # Sign + verify HMAC-SHA256
│   │   └── token.go                # Payment token generation (UUID)
│   ├── queue/
│   │   ├── client.go               # Asynq client
│   │   ├── worker.go               # Background job processor
│   │   └── tasks/
│   │       ├── notify_merchant.go  # Send webhook to merchant site
│   │       ├── expire_orders.go    # Mark expired pending orders
│   │       └── retry_webhook.go
│   └── templates/
│       └── checkout.html           # Payment checkout page
├── migrations/
│   ├── 000001_create_merchants.up.sql
│   ├── 000001_create_merchants.down.sql
│   ├── 000002_create_orders.up.sql
│   ├── 000002_create_orders.down.sql
│   ├── 000003_create_webhook_logs.up.sql
│   ├── 000003_create_webhook_logs.down.sql
│   ├── 000004_create_refunds.up.sql
│   ├── 000004_create_refunds.down.sql
│   ├── 000005_create_admin_users.up.sql
│   └── 000005_create_admin_users.down.sql
├── sql/
│   └── queries.sql                 # sqlc query definitions
├── sqlc.yaml
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
└── .env.example
```

### `payment-hub-admin/` (Next.js Dashboard)

```
payment-hub-admin/
├── app/
│   ├── layout.tsx
│   ├── page.tsx                    # Redirect to /dashboard
│   ├── login/page.tsx
│   ├── dashboard/
│   │   ├── page.tsx                # Overview stats
│   │   ├── transactions/page.tsx   # Transaction list + filters
│   │   ├── merchants/page.tsx      # Merchant management
│   │   ├── refunds/page.tsx
│   │   └── webhooks/page.tsx       # Webhook delivery logs
│   └── api/                        # Optional BFF routes
├── components/
│   ├── ui/                         # shadcn components
│   ├── stats-cards.tsx
│   ├── transaction-table.tsx
│   ├── domain-breakdown-chart.tsx
│   └── sidebar.tsx
├── lib/
│   ├── api.ts                      # Hub backend API client
│   └── auth.ts
├── package.json
└── .env.example
```

### `payment-hub-sdk-php/` (aMember Plugin SDK)

```
payment-hub-sdk-php/
├── src/
│   ├── BuyahrefClient.php          # HTTP client + signing
│   └── WebhookVerifier.php         # Verify incoming Hub webhooks
├── amember-plugin/
│   └── buyahref.php                # aMember payment plugin
├── composer.json
└── README.md
```

---

## 5. Database Schema (MySQL 8.0+)

> Production: existing VPS MySQL + phpMyAdmin. Local dev: Docker MySQL (see `docker-compose.yml`).
> UUIDs generated in Go application layer (`CHAR(36)`).

### `merchants`
```sql
CREATE TABLE merchants (
    id              CHAR(36) PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    domain          VARCHAR(255) NOT NULL UNIQUE,
    api_key         VARCHAR(64) NOT NULL UNIQUE,
    api_secret      VARCHAR(255) NOT NULL,
    webhook_url     VARCHAR(500) NOT NULL,
    return_url      VARCHAR(500) NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### `admin_users`
```sql
CREATE TABLE admin_users (
    id              CHAR(36) PRIMARY KEY,
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    name            VARCHAR(100) NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'admin',
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### `orders`
```sql
CREATE TABLE orders (
    id                  CHAR(36) PRIMARY KEY,
    hub_order_id        VARCHAR(20) NOT NULL UNIQUE,
    merchant_id         CHAR(36) NOT NULL,
    merchant_order_id   VARCHAR(100) NOT NULL,
    payment_token       VARCHAR(64) NOT NULL UNIQUE,
    amount              DECIMAL(12,2) NOT NULL,
    currency            VARCHAR(3) NOT NULL DEFAULT 'INR',
    status              VARCHAR(20) NOT NULL DEFAULT 'pending',
    customer_email      VARCHAR(255) NULL,
    customer_name       VARCHAR(255) NULL,
    customer_phone      VARCHAR(20) NULL,
    product_name        VARCHAR(255) NULL,
    product_description TEXT NULL,
    return_url          VARCHAR(500) NOT NULL,
    webhook_url         VARCHAR(500) NULL,
    phonepe_txn_id      VARCHAR(100) NULL,
    phonepe_response    JSON NULL,
    paid_at             DATETIME(3) NULL,
    expires_at          DATETIME(3) NOT NULL,
    created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_merchant_order (merchant_id, merchant_order_id),
    KEY idx_orders_merchant_status (merchant_id, status, created_at),
    KEY idx_orders_payment_token (payment_token),
    KEY idx_orders_status_expires (status, expires_at),
    CONSTRAINT fk_orders_merchant FOREIGN KEY (merchant_id) REFERENCES merchants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### `webhook_logs`
```sql
CREATE TABLE webhook_logs (
    id              CHAR(36) PRIMARY KEY,
    order_id        CHAR(36) NOT NULL,
    merchant_id     CHAR(36) NOT NULL,
    direction       VARCHAR(10) NOT NULL,
    payload         JSON NOT NULL,
    response_code   INT NULL,
    response_body   TEXT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    retry_count     INT NOT NULL DEFAULT 0,
    next_retry_at   DATETIME(3) NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_webhook_logs_order (order_id),
    KEY idx_webhook_logs_retry (status, next_retry_at),
    CONSTRAINT fk_webhook_logs_order FOREIGN KEY (order_id) REFERENCES orders(id),
    CONSTRAINT fk_webhook_logs_merchant FOREIGN KEY (merchant_id) REFERENCES merchants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### `refunds`
```sql
CREATE TABLE refunds (
    id                  CHAR(36) PRIMARY KEY,
    order_id            CHAR(36) NOT NULL,
    merchant_id         CHAR(36) NOT NULL,
    amount              DECIMAL(12,2) NOT NULL,
    reason              TEXT NULL,
    phonepe_refund_id   VARCHAR(100) NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'pending',
    initiated_by        CHAR(36) NULL,
    created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_refunds_order FOREIGN KEY (order_id) REFERENCES orders(id),
    CONSTRAINT fk_refunds_merchant FOREIGN KEY (merchant_id) REFERENCES merchants(id),
    CONSTRAINT fk_refunds_admin FOREIGN KEY (initiated_by) REFERENCES admin_users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### VPS MySQL Setup (Production)
```sql
CREATE DATABASE payment_hub CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'payment_hub'@'localhost' IDENTIFIED BY 'strong_password_here';
GRANT ALL PRIVILEGES ON payment_hub.* TO 'payment_hub'@'localhost';
FLUSH PRIVILEGES;
```

---

## 6. API Specification

Base URL: `https://buyahref.com`

### Authentication (All Merchant API calls)

Every request must include:
```
X-Merchant-Key:  mk_semrushtoolz_a1b2c3d4
X-Timestamp:     1717654321          (Unix timestamp, max 5 min old)
X-Signature:     HMAC-SHA256(secret, timestamp + method + path + body)
Content-Type:    application/json
```

**Signature generation:**
```
message = timestamp + "|" + method + "|" + path + "|" + body
signature = HMAC-SHA256(api_secret, message) → hex string
```

---

### 6.1 Create Order

```
POST /api/v1/orders/create
```

**Request:**
```json
{
  "order_id": "INV-12345",
  "amount": 999.00,
  "currency": "INR",
  "customer": {
    "email": "user@example.com",
    "name": "Rahul Sharma",
    "phone": "9876543210"
  },
  "product": {
    "name": "Premium Plan",
    "description": "1 Year Access"
  },
  "return_url": "https://semrushtoolz.com/amember/payment/return",
  "webhook_url": "https://semrushtoolz.com/amember/payment/webhook"
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "hub_order_id": "BH-000001",
    "payment_url": "https://buyahref.com/pay/abc123def456",
    "expires_at": "2025-06-06T16:00:00Z"
  }
}
```

**Errors:**
| Code | Reason |
|------|--------|
| 400 | Invalid payload / amount <= 0 |
| 401 | Invalid signature or expired timestamp |
| 409 | Duplicate order_id for this merchant |
| 429 | Rate limit exceeded |

---

### 6.2 Verify Order

```
GET /api/v1/orders/{merchant_order_id}/verify
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "hub_order_id": "BH-000001",
    "order_id": "INV-12345",
    "status": "success",
    "amount": 999.00,
    "currency": "INR",
    "phonepe_txn_id": "PP123456789",
    "paid_at": "2025-06-06T15:25:00Z"
  }
}
```

**Rule for merchants:** Only fulfill order if `status == "success"`. Never trust redirect URL alone.

---

### 6.3 Refund Order

```
POST /api/v1/orders/{merchant_order_id}/refund
```

**Request:**
```json
{
  "amount": 999.00,
  "reason": "Customer request"
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "refund_id": "RF-000001",
    "status": "pending"
  }
}
```

---

### 6.4 Checkout Page (Browser)

```
GET /pay/{payment_token}
```

- Public endpoint (no API key needed — token is secret).
- Shows: product name, amount, merchant name, "Pay with PhonePe" button.
- Button triggers PhonePe payment initiation.
- PhonePe redirect URL and callback URL both use `buyahref.com`.

---

### 6.5 PhonePe Webhook (Inbound)

```
POST /webhooks/phonepe
```

- Called by PhonePe servers only.
- Verify `X-VERIFY` header using PhonePe salt key.
- Update order status.
- Queue outbound webhook to merchant site.

---

### 6.6 Outbound Webhook (Hub → Merchant)

Hub POSTs to merchant's `webhook_url`:

```json
{
  "event": "payment.success",
  "hub_order_id": "BH-000001",
  "order_id": "INV-12345",
  "amount": 999.00,
  "currency": "INR",
  "status": "success",
  "phonepe_txn_id": "PP123456789",
  "paid_at": "2025-06-06T15:25:00Z",
  "timestamp": 1717654321,
  "signature": "HMAC-SHA256(...)"
}
```

**Events:** `payment.success` | `payment.failed` | `payment.refunded`

**Retry policy:** 3 retries — 30s, 5min, 30min. Log all attempts.

---

### 6.7 Admin API Endpoints

```
POST   /admin/api/auth/login
GET    /admin/api/stats/overview?from=&to=
GET    /admin/api/stats/by-merchant?from=&to=
GET    /admin/api/transactions?page=&limit=&merchant=&status=&from=&to=
GET    /admin/api/transactions/:hub_order_id
POST   /admin/api/refunds
GET    /admin/api/merchants
POST   /admin/api/merchants
PUT    /admin/api/merchants/:id
GET    /admin/api/webhook-logs?order_id=
GET    /admin/api/transactions/export?format=csv&from=&to=
```

---

## 7. Payment Flow (Step-by-Step)

### Happy Path — Success

```
STEP 1: User on semrushtoolz.com
        → Selects product, clicks "Pay with PhonePe"

STEP 2: aMember plugin (server-side)
        → POST buyahref.com/api/v1/orders/create
        → Receives payment_url

STEP 3: User redirected to
        → buyahref.com/pay/{token}
        → Sees: "Pay ₹999 for Premium Plan (semrushtoolz.com)"
        → Clicks "Pay with PhonePe"

STEP 4: Go backend
        → Calls PhonePe Create Payment API
        → Redirects user to PhonePe payment page
        → Order status: pending → processing

STEP 5: User completes payment on PhonePe

STEP 6: PhonePe sends webhook to
        → buyahref.com/webhooks/phonepe
        → Go verifies X-VERIFY signature
        → Order status: processing → success
        → Queues outbound webhook job

STEP 7: Background worker
        → POST semrushtoolz.com/amember/payment/webhook
        → aMember plugin verifies signature
        → Activates membership

STEP 8: PhonePe redirects user to
        → buyahref.com/pay/{token}/return
        → Go redirects to return_url:
        → semrushtoolz.com/amember/payment/return?order_id=INV-12345

STEP 9: aMember return handler
        → GET buyahref.com/api/v1/orders/INV-12345/verify
        → Confirms status = success
        → Shows "Payment Successful" page
```

### Failure Path

```
Steps 1-4: Same as above
Step 5: User cancels or payment fails on PhonePe
Step 6: PhonePe webhook → status = failed
Step 7: Outbound webhook → merchant notified of failure
Step 8: User redirected → semrushtoolz.com/return?status=failed
Step 9: aMember shows "Payment Failed, try again"
```

### Expiry Path

```
Step 1-3: Order created, user lands on checkout page
Step 4: User closes browser without paying
Step 5: After 30 minutes → cron job sets status = expired
Step 6: If user returns to /pay/{token} → "Payment link expired"
```

---

## 8. Security Specification

### 8.1 Request Authentication
- [ ] HMAC-SHA256 on every merchant API call
- [ ] Timestamp max 5 minutes old (prevent replay)
- [ ] API key identifies merchant — can only access own orders
- [ ] Invalid signature → 401, no data leaked

### 8.2 Amount Integrity
- [ ] Amount set server-side at order creation — never from URL params
- [ ] Verify API returns same amount merchant sent
- [ ] PhonePe charged amount must match DB amount before marking success

### 8.3 Payment Token
- [ ] UUID v4 — cryptographically random
- [ ] Single use — invalidated after success/fail/expiry
- [ ] Bound to specific order — cannot be reused for different order

### 8.4 PhonePe Webhook
- [ ] Verify X-VERIFY header on every inbound webhook
- [ ] Reject if signature invalid — log attempt
- [ ] Idempotent processing — duplicate webhook ignored if order already final

### 8.5 Outbound Webhook
- [ ] Hub signs every outbound webhook with merchant's secret
- [ ] Merchant MUST verify signature before fulfilling order
- [ ] Include timestamp — reject if > 5 min old

### 8.6 Rate Limiting (Redis)
```
Per merchant API key:  100 requests/minute
Per IP address:         20 requests/minute
Per payment token:       5 checkout page loads/minute
```

### 8.7 Database
- [ ] Parameterized queries only (sqlc — no raw string SQL)
- [ ] Row-level lock (SELECT FOR UPDATE) during status transitions
- [ ] Unique constraint on (merchant_id, merchant_order_id)

### 8.8 General
- [ ] HTTPS everywhere — no HTTP
- [ ] Secrets in environment variables only
- [ ] Admin passwords bcrypt hashed
- [ ] CORS — only allow registered merchant domains
- [ ] No sensitive data in logs (mask API secrets, card numbers)
- [ ] Checkout page — no iframe embedding (X-Frame-Options: DENY)

### 8.9 Fake Payment Prevention Checklist
```
Before activating membership on merchant site:
  ✅ Outbound webhook received with valid signature
  ✅ Verify API called — status = "success"
  ✅ Amount in webhook matches expected amount
  ✅ order_id matches the invoice being fulfilled
  ❌ NEVER trust redirect URL query params alone
  ❌ NEVER trust client-side JavaScript confirmation alone
```

---

## 9. PhonePe Integration

### Configuration (Environment)
```
PHONEPE_MERCHANT_ID=YOUR_MERCHANT_ID
PHONEPE_SALT_KEY=YOUR_SALT_KEY
PHONEPE_SALT_INDEX=1
PHONEPE_ENV=PRODUCTION          # or UAT for testing
PHONEPE_CALLBACK_URL=https://buyahref.com/webhooks/phonepe
PHONEPE_REDIRECT_URL=https://buyahref.com/pay/{token}/return
```

### PhonePe API Calls (Go client)

**1. Create Payment**
```
POST https://api.phonepe.com/apis/hermes/pg/v1/pay

Payload (base64 encoded):
{
  "merchantId": "...",
  "merchantTransactionId": "BH-000001",
  "merchantUserId": "user@example.com",
  "amount": 99900,               ← paise (₹999 = 99900)
  "redirectUrl": "https://buyahref.com/pay/{token}/return",
  "redirectMode": "POST",
  "callbackUrl": "https://buyahref.com/webhooks/phonepe",
  "paymentInstrument": { "type": "PAY_PAGE" }
}

Header: X-VERIFY = SHA256(base64payload + /pg/v1/pay + saltKey) + ### + saltIndex
```

**2. Check Status**
```
GET https://api.phonepe.com/apis/hermes/pg/v1/status/{merchantId}/{merchantTransactionId}
```

**3. Initiate Refund**
```
POST https://api.phonepe.com/apis/hermes/pg/v1/refund
```

**4. Webhook Verification**
```
X-VERIFY header = SHA256(base64responseBody + saltKey) + ### + saltIndex
Compare with received X-VERIFY — must match exactly
```

### Important PhonePe Rules
- `merchantTransactionId` = our `hub_order_id` (BH-000001)
- Amount always in **paise** (multiply by 100)
- Callback URL must be `buyahref.com` — PhonePe will reject others
- Redirect URL must be `buyahref.com` — then we redirect to merchant

---

## 10. Admin Dashboard Plan

### Pages

#### `/dashboard` — Overview
- Today's revenue, success count, failed count
- This month total
- Success rate percentage
- Per-domain breakdown table
- Revenue chart (last 30 days)

#### `/dashboard/transactions` — All Transactions
- Searchable, filterable table
- Filters: merchant, status, date range, amount range
- Columns: Hub ID, Merchant, Domain, Amount, Status, PhonePe TXN, Date
- Click row → transaction detail
- Export CSV button

#### `/dashboard/merchants` — Merchant Management
- List all registered sites
- Add new merchant → generates API key + secret (show once)
- Suspend / activate merchant
- View merchant-specific stats

#### `/dashboard/refunds` — Refunds
- Initiate refund (select transaction → amount → reason)
- Refund history with status

#### `/dashboard/webhooks` — Webhook Logs
- All inbound (PhonePe) and outbound (merchant) webhook logs
- Failed deliveries with retry button
- Payload viewer

### Dashboard Stats API Response Example
```json
{
  "overview": {
    "total_revenue": 45230.00,
    "success_count": 142,
    "failed_count": 8,
    "refunded_count": 3,
    "success_rate": 94.67
  },
  "by_merchant": [
    {
      "merchant_id": "...",
      "name": "semrushtoolz",
      "domain": "semrushtoolz.com",
      "revenue": 28400.00,
      "success_count": 89,
      "failed_count": 4
    }
  ]
}
```

---

## 11. Merchant Integration Guides

### 11.1 aMember Pro Plugin (PHP) — Now

**Installation:**
1. Copy `buyahref.php` to `amember/library/Am/Paysystem/`
2. Copy `BuyahrefClient.php` to same directory
3. aMember Admin → Setup → Payment Plugins → Enable "Buyahref Payment"

**Configuration in aMember Admin:**
```
Merchant API Key:    mk_semrushtoolz_xxxx
Merchant API Secret: secret_xxxx
Hub URL:             https://buyahref.com
```

**Plugin flow (buyahref.php):**
```php
class Am_Paysystem_Buyahref extends Am_Paysystem_Abstract {
    
    // Step 1: User clicks pay
    function _process($invoice, $request, $result) {
        $client = new BuyahrefClient($this->getConfig('api_key'), $this->getConfig('api_secret'));
        
        $response = $client->createOrder([
            'order_id'   => $invoice->public_id,
            'amount'     => $invoice->first_total,
            'customer'   => ['email' => $invoice->getEmail()],
            'product'    => ['name' => $invoice->getLineDescription()],
            'return_url' => $this->getReturnUrl(),
            'webhook_url'=> $this->getWebhookUrl(),
        ]);
        
        $result->setRedirect($response['payment_url']);
    }
    
    // Step 2: User returns after payment
    function _processValidated($invoice, $request, $result) {
        $client = new BuyahrefClient(...);
        $status = $client->verifyOrder($invoice->public_id);
        
        if ($status['status'] !== 'success') {
            throw new Am_Exception_Paysystem('Payment not confirmed');
        }
        // aMember activates membership automatically
    }
    
    // Step 3: Server webhook (most reliable)
    function directAction($request, $response, $invokeArgs) {
        $payload = json_decode(file_get_contents('php://input'), true);
        WebhookVerifier::verify($payload, $this->getConfig('api_secret'));
        
        if ($payload['event'] === 'payment.success') {
            $invoice = $this->getDi()->invoiceTable->findByPublicId($payload['order_id']);
            $invoice->addPayment(...);  // activate membership
        }
    }
}
```

---

### 11.2 Next.js Custom Dashboard — Future

**Install SDK (create `lib/buyahref.ts`):**
```typescript
export class BuyahrefClient {
  constructor(private apiKey: string, private secret: string, private hubUrl: string) {}

  async createOrder(params: CreateOrderParams): Promise<CreateOrderResponse> {
    const body = JSON.stringify(params);
    const timestamp = Math.floor(Date.now() / 1000).toString();
    const signature = this.sign(timestamp, 'POST', '/api/v1/orders/create', body);

    const res = await fetch(`${this.hubUrl}/api/v1/orders/create`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Merchant-Key': this.apiKey,
        'X-Timestamp': timestamp,
        'X-Signature': signature,
      },
      body,
    });
    return res.json();
  }

  async verifyOrder(orderId: string): Promise<VerifyOrderResponse> { ... }

  verifyWebhook(payload: WebhookPayload, signature: string): boolean { ... }

  private sign(timestamp: string, method: string, path: string, body: string): string {
    const message = `${timestamp}|${method}|${path}|${body}`;
    return createHmac('sha256', this.secret).update(message).digest('hex');
  }
}
```

**Next.js API Route — `/app/api/payment/create/route.ts`:**
```typescript
export async function POST(req: Request) {
  const { orderId, amount, customerEmail, productName } = await req.json();
  
  const hub = new BuyahrefClient(process.env.HUB_API_KEY!, process.env.HUB_SECRET!, process.env.HUB_URL!);
  
  const result = await hub.createOrder({
    order_id: orderId,
    amount,
    customer: { email: customerEmail },
    product: { name: productName },
    return_url: `${process.env.APP_URL}/payment/return`,
    webhook_url: `${process.env.APP_URL}/api/payment/webhook`,
  });

  return Response.json({ payment_url: result.data.payment_url });
}
```

**Next.js Webhook — `/app/api/payment/webhook/route.ts`:**
```typescript
export async function POST(req: Request) {
  const payload = await req.json();
  const signature = req.headers.get('X-Hub-Signature')!;

  const hub = new BuyahrefClient(...);
  if (!hub.verifyWebhook(payload, signature)) {
    return Response.json({ error: 'Invalid signature' }, { status: 401 });
  }

  if (payload.event === 'payment.success') {
    await db.orders.update({ id: payload.order_id, status: 'paid' });
    await activateSubscription(payload.order_id);
  }

  return Response.json({ received: true });
}
```

---

## 12. Implementation Phases

### Phase 1 — Foundation (Week 1)
**Goal:** Go project running with DB, basic API skeleton

| Task | Details | Done |
|------|---------|------|
| 1.1 | Initialize Go module, folder structure | [x] |
| 1.2 | Docker Compose: MySQL + Redis (local dev; prod uses VPS MySQL) | [x] |
| 1.3 | Database migrations (all 5 tables) | [x] |
| 1.4 | sqlc setup + basic queries | [x] |
| 1.5 | Config loading (Viper + .env) | [x] |
| 1.6 | Fiber server + health check endpoint | [x] |
| 1.7 | Logger setup (Zap) | [x] |

**Deliverable:** `GET /health` returns 200, DB connected

---

### Phase 2 — Core Order API (Week 2)
**Goal:** Create order + verify order working (PhonePe stub)

| Task | Details | Done |
|------|---------|------|
| 2.1 | HMAC signing library (`internal/security/hmac.go`) | [ ] |
| 2.2 | Auth middleware (verify signature + timestamp) | [ ] |
| 2.3 | Rate limit middleware (Redis) | [ ] |
| 2.4 | `POST /api/v1/orders/create` handler | [ ] |
| 2.5 | Order state machine in service layer | [ ] |
| 2.6 | Payment token generation (UUID) | [ ] |
| 2.7 | `GET /api/v1/orders/:id/verify` handler | [ ] |
| 2.8 | Duplicate order prevention (409) | [ ] |
| 2.9 | Unit tests for HMAC + state machine | [ ] |

**Deliverable:** Can create and verify orders via API (Postman/curl)

---

### Phase 3 — PhonePe Integration (Week 3)
**Goal:** Real PhonePe payment flow working end-to-end

| Task | Details | Done |
|------|---------|------|
| 3.1 | PhonePe client (`internal/phonepe/client.go`) | [ ] |
| 3.2 | PhonePe webhook verification | [ ] |
| 3.3 | Checkout page HTML template (`/pay/{token}`) | [ ] |
| 3.4 | PhonePe payment initiation from checkout page | [ ] |
| 3.5 | `POST /webhooks/phonepe` handler | [ ] |
| 3.6 | Return URL handler (`/pay/{token}/return`) | [ ] |
| 3.7 | PhonePe status check API (backup verification) | [ ] |
| 3.8 | Test with PhonePe UAT/sandbox environment | [ ] |

**Deliverable:** Can complete a real test payment on buyahref.com

---

### Phase 4 — Webhook System (Week 3-4)
**Goal:** Reliable outbound webhooks to merchant sites

| Task | Details | Done |
|------|---------|------|
| 4.1 | Asynq queue setup | [ ] |
| 4.2 | `notify_merchant` background job | [ ] |
| 4.3 | Webhook signing (outbound HMAC) | [ ] |
| 4.4 | Retry logic: 30s → 5min → 30min | [ ] |
| 4.5 | Webhook logs in DB | [ ] |
| 4.6 | `expire_orders` cron job (every 5 min) | [ ] |
| 4.7 | Queue worker process (`cmd/worker/main.go`) | [ ] |

**Deliverable:** Merchant sites receive signed webhooks reliably

---

### Phase 5 — Admin API + Dashboard (Week 4-5)
**Goal:** Admin can see all transactions, manage merchants

| Task | Details | Done |
|------|---------|------|
| 5.1 | Admin auth (login, JWT/session) | [ ] |
| 5.2 | Stats overview API | [ ] |
| 5.3 | Per-merchant stats API | [ ] |
| 5.4 | Transaction list + filter API | [ ] |
| 5.5 | Merchant CRUD API (+ API key generation) | [ ] |
| 5.6 | Refund API | [ ] |
| 5.7 | CSV export API | [ ] |
| 5.8 | Next.js project setup + auth | [ ] |
| 5.9 | Dashboard overview page | [ ] |
| 5.10 | Transactions page + filters | [ ] |
| 5.11 | Merchants management page | [ ] |
| 5.12 | Refunds page | [ ] |
| 5.13 | Webhook logs page | [ ] |

**Deliverable:** Full admin dashboard live on buyahref.com/admin

---

### Phase 6 — aMember Plugin (Week 5)
**Goal:** semrushtoolz.com accepting payments via Hub

| Task | Details | Done |
|------|---------|------|
| 6.1 | `BuyahrefClient.php` — HTTP + HMAC signing | [ ] |
| 6.2 | `WebhookVerifier.php` | [ ] |
| 6.3 | `buyahref.php` aMember plugin | [ ] |
| 6.4 | Register semrushtoolz as merchant in Hub | [ ] |
| 6.5 | Install plugin on semrushtoolz aMember | [ ] |
| 6.6 | End-to-end test: product → pay → membership activated | [ ] |
| 6.7 | Test failure flow | [ ] |
| 6.8 | Test concurrent payments (2 users simultaneously) | [ ] |

**Deliverable:** semrushtoolz.com live with PhonePe via Hub

---

### Phase 7 — Multi-Merchant + Hardening (Week 6)
**Goal:** All sites live, production-ready

| Task | Details | Done |
|------|---------|------|
| 7.1 | Register remaining 3 merchant sites | [ ] |
| 7.2 | Install aMember plugin on each | [ ] |
| 7.3 | Test each site end-to-end | [ ] |
| 7.4 | Production PhonePe credentials | [ ] |
| 7.5 | Nginx config + SSL | [ ] |
| 7.6 | Docker production deployment | [ ] |
| 7.7 | Monitoring (health check alerts) | [ ] |
| 7.8 | Load test: 50 concurrent payments | [ ] |
| 7.9 | Security review checklist (Section 8) | [ ] |
| 7.10 | Write Next.js integration docs (for future custom dashboard) | [ ] |

**Deliverable:** All sites live, production-ready system

---

## 13. Testing Checklist

### Unit Tests
- [ ] HMAC sign + verify (valid signature accepted)
- [ ] HMAC verify rejects tampered payload
- [ ] HMAC verify rejects expired timestamp (> 5 min)
- [ ] Order state machine: valid transitions allowed
- [ ] Order state machine: invalid transitions rejected
- [ ] Duplicate order_id returns 409
- [ ] PhonePe X-VERIFY validation (valid + invalid)

### Integration Tests
- [ ] Create order → returns payment_url
- [ ] Verify pending order → status = pending
- [ ] PhonePe webhook success → order status = success
- [ ] PhonePe webhook failed → order status = failed
- [ ] Outbound webhook delivered to merchant
- [ ] Outbound webhook retried on failure
- [ ] Order expires after 30 min
- [ ] Refund initiated → PhonePe refund API called

### End-to-End Tests
- [ ] Full payment on semrushtoolz → membership activated
- [ ] Payment failure → user sees fail page, no membership
- [ ] Payment cancel → user returns, can retry
- [ ] 2 simultaneous payments same site → both succeed independently
- [ ] 2 simultaneous payments different sites → both succeed independently
- [ ] Fake webhook (invalid signature) → rejected, order unchanged
- [ ] Verify API called before membership activation

### Admin Dashboard Tests
- [ ] Login with valid/invalid credentials
- [ ] Overview stats match DB counts
- [ ] Per-merchant filter works
- [ ] CSV export downloads correctly
- [ ] New merchant created → API key shown once
- [ ] Refund initiated from dashboard → order status = refunded

---

## 14. Deployment Guide

### Server Requirements (VPS)
```
OS:       Ubuntu 22.04 LTS
RAM:      2GB minimum (4GB recommended)
CPU:      2 cores
Storage:  20GB SSD
Ports:    80, 443 (Nginx), 8080 internal
```

### Docker Compose (Production)

```yaml
# docker-compose.prod.yml
# Production: MySQL runs on VPS host (not in Docker). Only app + redis here.
services:
  app:
    build: .
    restart: always
    env_file: .env
    network_mode: host
    depends_on: [redis]

  worker:
    build: .
    command: ["./worker"]
    restart: always
    env_file: .env
    network_mode: host
    depends_on: [redis]

  redis:
    image: redis:7-alpine
    restart: always
    volumes: ["redisdata:/data"]

volumes:
  redisdata:
```

### Nginx Config

```nginx
# buyahref.com
server {
    listen 443 ssl;
    server_name buyahref.com;

    ssl_certificate     /etc/letsencrypt/live/buyahref.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/buyahref.com/privkey.pem;

    # Go API + Checkout
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    # Next.js Admin Dashboard
    location /admin {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
    }
}
```

### Deployment Steps
```bash
# 1. Clone repo on VPS
git clone ... /opt/payment-hub
cd /opt/payment-hub

# 2. Configure environment
cp .env.example .env
nano .env   # fill in all values

# 3. Run migrations
make migrate-up

# 4. Build and start
docker compose -f docker-compose.prod.yml up -d --build

# 5. Create admin user
make create-admin EMAIL=admin@buyahref.com

# 6. Verify
curl https://buyahref.com/health
```

---

## 15. Environment Variables

```bash
# .env.example

# Server
APP_ENV=production
APP_PORT=8080
APP_URL=https://buyahref.com

# Database
DB_HOST=127.0.0.1
DB_PORT=5432
DB_NAME=payment_hub
DB_USER=hub
DB_PASSWORD=change_me_strong_password

# Redis
REDIS_URL=redis://redis:6379

# PhonePe
PHONEPE_MERCHANT_ID=YOUR_MERCHANT_ID
PHONEPE_SALT_KEY=YOUR_SALT_KEY
PHONEPE_SALT_INDEX=1
PHONEPE_ENV=PRODUCTION

# Security
JWT_SECRET=change_me_random_64_chars
ADMIN_JWT_EXPIRY=24h
ORDER_EXPIRY_MINUTES=30

# Admin
ADMIN_EMAIL=admin@buyahref.com
ADMIN_PASSWORD=change_me_on_first_login

# Logging
LOG_LEVEL=info
```

---

## 16. Future Roadmap

### Next Month — Custom Dashboard (semrushtoolz)
- Build Next.js dashboard on semrushtoolz.com
- Use same Hub API (no backend changes)
- aMember gradually replaced
- `lib/buyahref.ts` SDK copied from plan (Section 11.2)

### Future Features (Backlog)
| Feature | Priority |
|---------|----------|
| Multiple payment methods (Razorpay backup) | Medium |
| Partial refunds | Medium |
| Email notifications on payment | Low |
| Merchant self-service portal (view own stats) | Low |
| Payment links (no integration needed) | Low |
| Subscription/recurring payments | Low |
| API rate limit dashboard | Low |
| Webhook delivery dashboard with manual retry | Medium |

---

## Quick Reference — Key URLs

| URL | Purpose |
|-----|---------|
| `https://buyahref.com/health` | Health check |
| `https://buyahref.com/api/v1/orders/create` | Create payment order |
| `https://buyahref.com/api/v1/orders/{id}/verify` | Verify payment status |
| `https://buyahref.com/pay/{token}` | Checkout page (user-facing) |
| `https://buyahref.com/webhooks/phonepe` | PhonePe callback |
| `https://buyahref.com/admin` | Admin dashboard |

---

## Quick Reference — Merchant Onboarding Checklist

When adding a new merchant site:

```
[ ] Create merchant in admin dashboard
[ ] Note API key + secret (shown once — save securely)
[ ] Set webhook_url and return_url
[ ] Install aMember plugin (or integrate API client)
[ ] Configure plugin with API key + secret
[ ] Test with ₹1 payment
[ ] Verify membership activates on success
[ ] Verify failure flow works
[ ] Add to monitoring
```

---

*Plan Version: 1.0 — Ready for implementation*
*Start with Phase 1: Go project setup + Docker + Database*
