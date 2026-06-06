package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/buyahref/payment-hub/internal/models"
	"github.com/buyahref/payment-hub/internal/order"
)

func (r *OrderRepository) GetByPaymentToken(ctx context.Context, token string) (*models.Order, error) {
	const q = `
		SELECT id, hub_order_id, merchant_id, merchant_order_id, payment_token,
		       amount, currency, status,
		       COALESCE(customer_email, ''), COALESCE(customer_name, ''), COALESCE(customer_phone, ''),
		       COALESCE(product_name, ''), COALESCE(product_description, ''),
		       return_url, COALESCE(webhook_url, ''), COALESCE(phonepe_txn_id, ''),
		       paid_at, expires_at, created_at, updated_at
		FROM orders
		WHERE payment_token = ?
		LIMIT 1
	`
	return r.scanOrder(r.db.QueryRowContext(ctx, q, token))
}

func (r *OrderRepository) GetByHubOrderID(ctx context.Context, hubOrderID string) (*models.Order, error) {
	const q = `
		SELECT id, hub_order_id, merchant_id, merchant_order_id, payment_token,
		       amount, currency, status,
		       COALESCE(customer_email, ''), COALESCE(customer_name, ''), COALESCE(customer_phone, ''),
		       COALESCE(product_name, ''), COALESCE(product_description, ''),
		       return_url, COALESCE(webhook_url, ''), COALESCE(phonepe_txn_id, ''),
		       paid_at, expires_at, created_at, updated_at
		FROM orders
		WHERE hub_order_id = ?
		LIMIT 1
	`
	return r.scanOrder(r.db.QueryRowContext(ctx, q, hubOrderID))
}

func (r *OrderRepository) TransitionStatus(ctx context.Context, orderID, fromStatus, toStatus string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders SET status = ?, updated_at = NOW(3)
		WHERE id = ? AND status = ?
	`, toStatus, orderID, fromStatus)
	if err != nil {
		return fmt.Errorf("transition status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *OrderRepository) MarkSuccess(ctx context.Context, orderID, phonepeTxnID string, phonepeResponse []byte) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET status = ?, phonepe_txn_id = ?, phonepe_response = ?, paid_at = NOW(3), updated_at = NOW(3)
		WHERE id = ? AND status IN (?, ?)
	`, models.OrderStatusSuccess, phonepeTxnID, nullJSON(phonepeResponse), orderID,
		models.OrderStatusPending, models.OrderStatusProcessing)
	if err != nil {
		return fmt.Errorf("mark success: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// idempotent: already success
		var status string
		err := r.db.QueryRowContext(ctx, `SELECT status FROM orders WHERE id = ?`, orderID).Scan(&status)
		if err == nil && status == models.OrderStatusSuccess {
			return nil
		}
		return ErrNotFound
	}
	return nil
}

func (r *OrderRepository) MarkFailed(ctx context.Context, orderID string, phonepeResponse []byte) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET status = ?, phonepe_response = ?, updated_at = NOW(3)
		WHERE id = ? AND status IN (?, ?)
	`, models.OrderStatusFailed, nullJSON(phonepeResponse), orderID,
		models.OrderStatusPending, models.OrderStatusProcessing)
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var status string
		err := r.db.QueryRowContext(ctx, `SELECT status FROM orders WHERE id = ?`, orderID).Scan(&status)
		if err == nil && order.IsFinalStatus(status) {
			return nil
		}
		return ErrNotFound
	}
	return nil
}

func (r *OrderRepository) SavePhonePeResponse(ctx context.Context, orderID string, phonepeResponse []byte) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE orders SET phonepe_response = ?, updated_at = NOW(3) WHERE id = ?
	`, nullJSON(phonepeResponse), orderID)
	return err
}

func (r *MerchantRepository) GetByID(ctx context.Context, id string) (*models.Merchant, error) {
	const q = `
		SELECT id, name, domain, api_key, api_secret, webhook_url,
		       COALESCE(return_url, ''), status, created_at, updated_at
		FROM merchants WHERE id = ? LIMIT 1
	`
	m := &models.Merchant{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&m.ID, &m.Name, &m.Domain, &m.APIKey, &m.APISecret,
		&m.WebhookURL, &m.ReturnURL, &m.Status, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get merchant: %w", err)
	}
	return m, nil
}

func (r *OrderRepository) scanOrder(row *sql.Row) (*models.Order, error) {
	o := &models.Order{}
	var paidAt sql.NullTime
	err := row.Scan(
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
		return nil, fmt.Errorf("scan order: %w", err)
	}
	if paidAt.Valid {
		o.PaidAt = &paidAt.Time
	}
	return o, nil
}

func nullJSON(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}

func (r *OrderRepository) ExpirePendingOrders(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders SET status = ?, updated_at = NOW(3)
		WHERE status = ? AND expires_at < ?
	`, models.OrderStatusExpired, models.OrderStatusPending, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
