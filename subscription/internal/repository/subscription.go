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

type Subscription interface {
	// Create creates a new subscription in the database.
	Create(ctx context.Context, subscription *domain.Subscription, transaction *domain.Transaction) error
	// PauseAndCreate pauses an active subscription and creates a new one.
	PauseAndCreate(ctx context.Context, pausedSub, newSub *domain.Subscription, transaction *domain.Transaction) error

	// GetAll retrieves all subscriptions by user id from the database.
	GetAll(ctx context.Context, userId uint64) ([]*domain.Subscription, error)
	// GetActive retrieves the active subscription.
	GetActive(ctx context.Context, userId uint64) (*domain.Subscription, error)

	// Update updates an existing subscription in the database.
	Update(ctx context.Context, id uuid.UUID, subscription *domain.Subscription) error
}

type subscriptionRepo struct {
	db *pgxpool.Pool
}

func (s *subscriptionRepo) Create(ctx context.Context, subscription *domain.Subscription, transaction *domain.Transaction) error {
	return postgres.RunInTx(ctx, s.db, func(tx pgx.Tx) error {
		if err := s.createInTx(ctx, tx, subscription); err != nil {
			return err
		}

		return s.updateTransactionStatusInTx(ctx, tx, transaction)
	})
}

func (s *subscriptionRepo) PauseAndCreate(ctx context.Context, pausedSub, newSub *domain.Subscription, transaction *domain.Transaction) error {
	return postgres.RunInTx(ctx, s.db, func(tx pgx.Tx) error {
		if pausedSub.UserID != newSub.UserID {
			return fmt.Errorf("cannot transfer subscription between users: %w", ErrInvalid)
		}

		if _, err := tx.Exec(ctx, `
			UPDATE subscriptions
			SET status = 'PAUSED', paused_at = NOW(), remaining_days = EXTRACT(DAY FROM (end_date - NOW()))
			WHERE id = $1 AND version = $2
		`, pausedSub.ID, pausedSub.Version); err != nil {
			var pgErr *pgconn.PgError
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				return fmt.Errorf("subscription %w: %w", ErrNotFound, err)
			case errors.As(err, &pgErr) && pgErr.Code == "23505":
				return fmt.Errorf("subscription already exists: %w", ErrDuplicate)
			default:
				return fmt.Errorf("%w: %w", ErrUnknown, err)
			}
		}

		if err := s.createInTx(ctx, tx, newSub); err != nil {
			return err
		}

		return s.updateTransactionStatusInTx(ctx, tx, transaction)
	})
}

func (s *subscriptionRepo) createInTx(ctx context.Context, tx pgx.Tx, subscription *domain.Subscription) error {
	if _, err := tx.Exec(ctx, `
			INSERT INTO subscriptions (id, user_id, plan_id, status, start_date)
			VALUES ($1, $2, $3, $4, $5)
		`, subscription.ID, subscription.UserID, subscription.PlanID, subscription.Status, subscription.StartDate); err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			return fmt.Errorf("subscription %w: %w", ErrDuplicate, err)
		default:
			return fmt.Errorf("%w: %w", ErrUnknown, err)
		}
	}

	return nil
}

func (s *subscriptionRepo) updateTransactionStatusInTx(ctx context.Context, tx pgx.Tx, transaction *domain.Transaction) error {
	if _, err := tx.Exec(ctx, `
			UPDATE transactions SET status = $1 WHERE id = $2
		`, transaction.Status.String(), transaction.ID,
	); err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("transaction %w: %w", ErrNotFound, err)
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			return fmt.Errorf("transaction already exists: %w", ErrDuplicate)
		default:
			return fmt.Errorf("%w: %w", ErrUnknown, err)
		}
	}

	return nil
}

func (s *subscriptionRepo) GetAll(ctx context.Context, userId uint64) ([]*domain.Subscription, error) {
	var subscriptions []*domain.Subscription

	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, plan_id, status, start_date, end_date, paused_at, remaining_days, version
		FROM subscriptions
		WHERE user_id = $1
	`, userId)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var subscription domain.Subscription
		if err = rows.Scan(
			&subscription.ID, &subscription.UserID, &subscription.PlanID, &subscription.Status, &subscription.StartDate,
			&subscription.EndDate, &subscription.PausedAt, &subscription.RemainingDays, &subscription.Version,
		); err != nil {
			return nil, err
		}

		subscriptions = append(subscriptions, &subscription)
	}

	return subscriptions, nil
}

func (s *subscriptionRepo) GetActive(ctx context.Context, userId uint64) (*domain.Subscription, error) {
	var subscription domain.Subscription
	if err := s.db.QueryRow(ctx, `
		SELECT id, user_id, plan_id, status, start_date, end_date, paused_at, remaining_days, version
		FROM subscriptions
		WHERE user_id = $1 AND status = 'ACTIVE'
	`, userId).Scan(
		&subscription.ID, &subscription.UserID, &subscription.PlanID, &subscription.Status, &subscription.StartDate,
		&subscription.EndDate, &subscription.PausedAt, &subscription.RemainingDays, &subscription.Version,
	); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, nil
		default:
			return nil, fmt.Errorf("%w: %w", ErrUnknown, err)
		}
	}

	return &subscription, nil
}

func (s *subscriptionRepo) Update(ctx context.Context, id uuid.UUID, subscription *domain.Subscription) error {
	if _, err := s.db.Exec(ctx, `
		UPDATE subscriptions
		SET status = $1, start_date = $2, end_date = $3, version = version + 1
		WHERE id = $4 AND version = $5
	`, subscription.Status, subscription.StartDate, subscription.EndDate, id, subscription.Version); err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("subscription %w: %w", ErrNotFound, err)
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			return fmt.Errorf("subscription already exists: %w", ErrDuplicate)
		default:
			return fmt.Errorf("%w: %w", ErrUnknown, err)
		}
	}

	return nil
}
