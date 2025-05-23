package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ziliscite/bard_narate/subscription/internal/domain"
	"github.com/ziliscite/bard_narate/subscription/pkg/postgres"
)

type DiscountValidator interface {
	// ValidateAndGet get discount plans and validates if the discount is applicable to the plan.
	ValidateAndGet(ctx context.Context, code string, planID uint64) (*domain.Discount, bool, error)
}

type DiscountWriter interface {
	// Create creates a new plan in the database.
	Create(ctx context.Context, discount *domain.Discount) error
	// Update updates an existing plan in the database.
	Update(ctx context.Context, discount *domain.Discount) error
	// AttachPlansToDiscount adds plans to a discount in batch operation
	AttachPlansToDiscount(ctx context.Context, discountID uint64, planIDS ...uint64) error
}

type Discount interface {
	DiscountValidator
	DiscountWriter
}

type discountRepo struct {
	db *pgxpool.Pool
}

func NewDiscountRepository(db *pgxpool.Pool) Discount {
	return &discountRepo{
		db: db,
	}
}

func (r *discountRepo) ValidateAndGet(ctx context.Context, code string, planID uint64) (*domain.Discount, bool, error) {
	query := `
		SELECT d.*
		FROM discounts d
		WHERE d.code = $1 AND (d.scope = 'ALL' OR (d.scope = 'PLANS' AND EXISTS (
			SELECT 1
			FROM discount_plans dp
			WHERE dp.discount_id = d.id AND dp.plan_id = $2
		)));
	`

	var discount domain.Discount
	if err := r.db.QueryRow(ctx, query, code, planID).Scan(
		&discount.ID, &discount.Code, &discount.Description, &discount.Scope,
		&discount.PercentageValue, &discount.StartDate, &discount.EndDate,
		&discount.CreatedAt, &discount.UpdatedAt, &discount.Version,
	); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, false, nil // No discount found for a code/plan combination
		default:
			return nil, false, err

		}
	}

	return &discount, true, nil
}

func (r *discountRepo) Create(ctx context.Context, discount *domain.Discount) error {
	if _, err := r.db.Exec(ctx, `
		INSERT INTO discounts (code, description, scope, percentage_value, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, discount.Code, discount.Description, discount.Scope, discount.PercentageValue, discount.StartDate, discount.EndDate,
	); err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			return fmt.Errorf("discount %w: %w", ErrDuplicate, err)
		default:
			return fmt.Errorf("%w: %w", ErrUnknown, err)
		}
	}

	return nil
}

func (r *discountRepo) Update(ctx context.Context, discount *domain.Discount) error {
	if _, err := r.db.Exec(ctx, `
		UPDATE discounts SET description = $1, scope = $2, percentage_value = $3, start_date = $4, end_date = $5
		WHERE id = $6
	`, discount.Description, discount.Scope, discount.PercentageValue, discount.StartDate, discount.EndDate, discount.ID,
	); err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("discount %w: %w", ErrNotFound, err)
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			return fmt.Errorf("discount already exists: %w", ErrDuplicate)
		default:
			return fmt.Errorf("%w: %w", ErrUnknown, err)
		}
	}

	return nil
}

func (r *discountRepo) AttachPlansToDiscount(ctx context.Context, discountID uint64, planIDS ...uint64) error {
	return postgres.RunInTx(ctx, r.db, func(tx pgx.Tx) error {
		var batch *pgx.Batch
		for _, planID := range planIDS {
			batch.Queue(`
				INSERT INTO discount_plans (discount_id, plan_id)
				VALUES ($1, $2)
				ON CONFLICT (discount_id, plan_id) DO NOTHING
            `, discountID, planID)
		}

		results := tx.SendBatch(ctx, batch)
		defer results.Close()

		for range planIDS {
			if _, err := results.Exec(); err != nil {
				return fmt.Errorf("failed to execute batch insert: %w", err)
			}
		}

		return nil
	})
}
