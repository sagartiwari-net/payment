-- Run this ONCE in phpMyAdmin → database "paymentsystem" → SQL tab
-- Order matters (foreign keys)

CREATE TABLE IF NOT EXISTS merchants (
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

CREATE TABLE IF NOT EXISTS admin_users (
    id              CHAR(36) PRIMARY KEY,
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    name            VARCHAR(100) NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'admin',
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS orders (
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

CREATE TABLE IF NOT EXISTS webhook_logs (
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

CREATE TABLE IF NOT EXISTS refunds (
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
