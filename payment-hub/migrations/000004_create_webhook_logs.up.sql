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
