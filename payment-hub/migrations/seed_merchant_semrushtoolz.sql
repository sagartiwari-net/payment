-- Seed merchant: semrushtoolz.com
-- Run once in phpMyAdmin → paymentsystem → SQL

INSERT INTO merchants (
    id, name, domain, api_key, api_secret,
    webhook_url, return_url, status
) VALUES (
    'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
    'semrushtoolz',
    'semrushtoolz.com',
    'mk_semrushtoolz_001',
    'sk_semrushtoolz_secret_change_me_in_production',
    'https://semrushtoolz.com/amember/payment/buyahref/webhook',
    'https://semrushtoolz.com/amember/payment/buyahref/return',
    'active'
) ON DUPLICATE KEY UPDATE name = VALUES(name);
