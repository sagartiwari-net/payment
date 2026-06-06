package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/buyahref/payment-hub/internal/models"
)

var ErrNotFound = errors.New("not found")
var ErrDuplicateOrder = errors.New("duplicate order")

type MerchantRepository struct {
	db *sql.DB
}

func NewMerchantRepository(db *sql.DB) *MerchantRepository {
	return &MerchantRepository{db: db}
}

func (r *MerchantRepository) GetByAPIKey(ctx context.Context, apiKey string) (*models.Merchant, error) {
	const q = `
		SELECT id, name, domain, api_key, api_secret, webhook_url,
		       COALESCE(return_url, ''), status, created_at, updated_at
		FROM merchants
		WHERE api_key = ?
		LIMIT 1
	`

	m := &models.Merchant{}
	err := r.db.QueryRowContext(ctx, q, apiKey).Scan(
		&m.ID, &m.Name, &m.Domain, &m.APIKey, &m.APISecret,
		&m.WebhookURL, &m.ReturnURL, &m.Status, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get merchant by api key: %w", err)
	}
	return m, nil
}

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, o *models.Order) error {
	const q = `
		INSERT INTO orders (
			id, hub_order_id, merchant_id, merchant_order_id, payment_token,
			amount, currency, status, customer_email, customer_name, customer_phone,
			product_name, product_description, return_url, webhook_url, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, q,
		o.ID, o.HubOrderID, o.MerchantID, o.MerchantOrderID, o.PaymentToken,
		o.Amount, o.Currency, o.Status, nullString(o.CustomerEmail), nullString(o.CustomerName),
		nullString(o.CustomerPhone), nullString(o.ProductName), nullString(o.ProductDescription),
		o.ReturnURL, nullString(o.WebhookURL), o.ExpiresAt,
	)
	if err != nil {
		if isDuplicateKey(err) {
			return ErrDuplicateOrder
		}
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}

func (r *OrderRepository) GetByMerchantOrderID(ctx context.Context, merchantID, merchantOrderID string) (*models.Order, error) {
	const q = `
		SELECT id, hub_order_id, merchant_id, merchant_order_id, payment_token,
		       amount, currency, status,
		       COALESCE(customer_email, ''), COALESCE(customer_name, ''), COALESCE(customer_phone, ''),
		       COALESCE(product_name, ''), COALESCE(product_description, ''),
		       return_url, COALESCE(webhook_url, ''), COALESCE(phonepe_txn_id, ''),
		       paid_at, expires_at, created_at, updated_at
		FROM orders
		WHERE merchant_id = ? AND merchant_order_id = ?
		LIMIT 1
	`

	o := &models.Order{}
	var paidAt sql.NullTime
	err := r.db.QueryRowContext(ctx, q, merchantID, merchantOrderID).Scan(
		&o.ID, &o.HubOrderID, &o.MerchantID, &o.MerchantOrderID, &o.PaymentToken,
		&o.Amount, &o.Currency, &o.Status,
		&o.CustomerEmail, &o.CustomerName, &o.CustomerPhone,
		&o.ProductName, &o.ProductDescription,
		&o.ReturnURL, &o.WebhookURL, &o.PhonePeTxnID,
		&paidAt, &o.ExpiresAt, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	if paidAt.Valid {
		o.PaidAt = &paidAt.Time
	}
	return o, nil
}

func (r *OrderRepository) AllocateHubOrderID(ctx context.Context) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SELECT GET_LOCK('hub_order_id', 10)"); err != nil {
		return "", fmt.Errorf("acquire lock: %w", err)
	}
	defer tx.ExecContext(ctx, "SELECT RELEASE_LOCK('hub_order_id')") //nolint:errcheck

	var maxNum sql.NullInt64
	err = tx.QueryRowContext(ctx, `
		SELECT MAX(CAST(SUBSTRING(hub_order_id, 4) AS UNSIGNED))
		FROM orders
		WHERE hub_order_id LIKE 'BH-%'
	`).Scan(&maxNum)
	if err != nil {
		return "", fmt.Errorf("max hub order id: %w", err)
	}

	next := int64(1)
	if maxNum.Valid {
		next = maxNum.Int64 + 1
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}

	return fmt.Sprintf("BH-%06d", next), nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func isDuplicateKey(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "1062"))
}
