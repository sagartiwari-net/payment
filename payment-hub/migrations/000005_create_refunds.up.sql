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
