package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ziliscite/bard_narate/subscription/internal/domain"
	"github.com/ziliscite/bard_narate/subscription/pkg/postgres"
)

type Transaction interface {
	New(ctx context.Context, userID uint64, planID uint64, currency string, price float64, calculate func(*domain.Transaction) error) (*domain.Transaction, error)
	GetTransaction(ctx context.Context, transactionID string) (*domain.Transaction, error)
	GetTransactionAndOrder(ctx context.Context, transactionID string) (*domain.Transaction, *domain.Order, error)
	GetOrderByTransactionID(ctx context.Context, transactionID string) (*domain.Order, error)
	Update(ctx context.Context, transactionID uuid.UUID, transaction *domain.Transaction) error
}

type transactionRepository struct {
	db *pgxpool.Pool
}

func NewPaymentRepository(db *pgxpool.Pool) Transaction {
	return &transactionRepository{
		db: db,
	}
}

func (p *transactionRepository) New(ctx context.Context, userID uint64, planID uint64, currency string, price float64, calculate func(*domain.Transaction) error) (*domain.Transaction, error) {
	return postgres.RunInTxReturn[domain.Transaction](ctx, p.db, func(tx pgx.Tx) (*domain.Transaction, error) {
		order, err := domain.NewOrder(userID, planID)
		if err != nil {
			return nil, err
		}

		if _, err = tx.Exec(ctx, `
			INSERT INTO orders (id, user_id, plan_id)
			VALUES ($1, $2, $3)
		`, order.ID, order.UserID, order.PlanID); err != nil {
			var pgErr *pgconn.PgError
			switch {
			case errors.As(err, &pgErr) && pgErr.Code == "23505":
				return nil, fmt.Errorf("order %w: %w", ErrDuplicate, err)
			default:
				return nil, fmt.Errorf("%w: %w", ErrUnknown, err)
			}
		}

		transaction, err := domain.NewTransaction(order.ID, price, currency)
		if err != nil {
			return nil, err
		}

		if err = calculate(transaction); err != nil {
			return nil, err
		}

		args := []interface{}{
			transaction.ID, transaction.OrderID, transaction.SubTotal, transaction.Tax, transaction.ProcessingFee,
			transaction.Discount, transaction.Total, transaction.Currency.String(), transaction.Status, transaction.IdempotencyKey,
		}

		if _, err = tx.Exec(ctx, `
			INSERT INTO transactions (id, order_id, subtotal, tax, processing_fee, discount, total, currency, status, idempotency_key)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, args...); err != nil {
			var pgErr *pgconn.PgError
			switch {
			case errors.As(err, &pgErr) && pgErr.Code == "23505":
				return nil, fmt.Errorf("subscription %w: %w", ErrDuplicate, err)
			default:
				return nil, fmt.Errorf("%w: %w", ErrUnknown, err)
			}
		}

		// later in the service layer,
		// call payment gateway and get token
		// use token and call gateway again with transaction detail and shit
		// get redirect url

		return transaction, nil
	})
}

func (p *transactionRepository) GetTransactionAndOrder(ctx context.Context, transactionID string) (*domain.Transaction, *domain.Order, error) {
	t, err := p.GetTransaction(ctx, transactionID)
	if err != nil {
		return nil, nil, err
	}

	var o domain.Order
	if err = p.db.QueryRow(ctx, `
		SELECT * FROM orders WHERE id = $1
	`, t.OrderID).Scan(
		&o.ID, &o.UserID, &o.PlanID, &o.CreatedAt,
	); err != nil {
		return nil, nil, err
	}

	return t, &o, nil
}

func (p *transactionRepository) GetTransaction(ctx context.Context, transactionID string) (*domain.Transaction, error) {
	var (
		t domain.Transaction
		c string
		s string
	)

	if err := p.db.QueryRow(ctx, `
		SELECT * FROM transactions WHERE id = $1
	`, transactionID).Scan(
		&t.ID, &t.OrderID, &t.SubTotal, &t.Tax,
		&t.ProcessingFee, &t.Discount, &t.Total,
		&c, &s, &t.IdempotencyKey,
		&t.CreatedAt, &t.UpdatedAt, &t.Version,
	); err != nil {
		return nil, err
	}

	currency, err := domain.NewCurrency(c)
	if err != nil {
		return nil, err
	}
	t.Currency = currency

	status, err := domain.ParsePaymentStatus(s)
	if err != nil {
		return nil, err
	}
	t.Status = status

	return &t, nil
}

func (p *transactionRepository) GetOrderByTransactionID(ctx context.Context, transactionID string) (*domain.Order, error) {
	var o domain.Order
	if err := p.db.QueryRow(ctx, `
		SELECT o.*
		FROM orders o
		INNER JOIN transactions t ON t.order_id = o.id
		WHERE t.id = $1
	`, transactionID).Scan(
		&o.ID, &o.UserID, &o.PlanID, &o.CreatedAt,
	); err != nil {
		return nil, err
	}

	return &o, nil
}

func (p *transactionRepository) Update(ctx context.Context, transactionID uuid.UUID, transaction *domain.Transaction) error {
	if _, err := p.db.Exec(ctx, `
		UPDATE transactions
		SET subtotal = $2, tax = $3, processing_fee = $4, discount = $5, 
		    total = $6, currency = $7, status = $8, idempotency_key = $9
		WHERE id = $1
	`, transactionID, transaction.SubTotal, transaction.Tax, transaction.ProcessingFee, transaction.Discount,
		transaction.Total, transaction.Currency.String(), transaction.Status, transaction.IdempotencyKey,
	); err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			return fmt.Errorf("transaction %w: %w", ErrDuplicate, err)
		default:
			return fmt.Errorf("%w: %w", ErrUnknown, err)
		}
	}

	return nil
}
