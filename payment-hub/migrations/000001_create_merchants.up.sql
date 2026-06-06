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
