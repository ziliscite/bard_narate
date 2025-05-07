package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ziliscite/bard_narate/subscription/internal/domain"
)

type PlanReader interface {
	// Get retrieves a plan by its ID from the database.
	Get(ctx context.Context, id uint64) (*domain.Plan, error)
	// GetAll retrieves all plans from the database.
	GetAll(ctx context.Context) ([]*domain.Plan, error)
}

type PlanWriter interface {
	// Create creates a new plan in the database.
	Create(ctx context.Context, plan *domain.Plan) error
	// Update updates plan in the database.
	Update(ctx context.Context, plan *domain.Plan) error
}

type Plan interface {
	PlanReader
	PlanWriter
}

type planRepo struct {
	db *pgxpool.Pool
}

func NewPlanRepository(db *pgxpool.Pool) Plan {
	return &planRepo{
		db: db,
	}
}

func (p *planRepo) Get(ctx context.Context, id uint64) (*domain.Plan, error) {
	var plan domain.Plan
	var cur string
	if err := p.db.QueryRow(ctx, `
		SELECT * FROM plans WHERE id = $1
	`, id).Scan(
		&plan.ID, &plan.Name, &plan.Description, &plan.Price, &cur,
		&plan.DurationDays, &plan.CreatedAt, &plan.UpdatedAt, &plan.Version,
	); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, nil
		default:
			return nil, fmt.Errorf("%w: %w", ErrUnknown, err)
		}
	}

	curr, err := domain.NewCurrency(cur)
	if err != nil {
		return nil, err
	}

	plan.Currency = curr
	return &plan, nil
}

func (p *planRepo) GetAll(ctx context.Context) ([]*domain.Plan, error) {
	var plans []*domain.Plan

	rows, err := p.db.Query(ctx, `SELECT * FROM plans`)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, fmt.Errorf("plans %w: %w", ErrNotFound, err)
		default:
			return nil, fmt.Errorf("%w: %w", ErrUnknown, err)
		}
	}

	for rows.Next() {
		var plan domain.Plan
		var cur string
		if err = rows.Scan(
			&plan.ID, &plan.Name, &plan.Description, &plan.Price, &cur,
			&plan.DurationDays, &plan.CreatedAt, &plan.UpdatedAt, &plan.Version,
		); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrUnknown, err)
		}

		curr, err := domain.NewCurrency(cur)
		if err != nil {
			return nil, err
		}

		plan.Currency = curr
		plans = append(plans, &plan)
	}

	return plans, nil
}

func (p *planRepo) Create(ctx context.Context, plan *domain.Plan) error {
	if _, err := p.db.Exec(ctx, `
		INSERT INTO plans (name, description, price, currency, duration_days)
		VALUES ($1, $2, $3, $4, $5)
	`, plan.Name, plan.Description, plan.Price, plan.Currency.String(), plan.DurationDays); err != nil {
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

func (p *planRepo) Update(ctx context.Context, plan *domain.Plan) error {
	if _, err := p.db.Exec(ctx, `
		UPDATE plans SET name = $1, description = $2, price = $3, currency = $4, duration_days = $5
		WHERE id = $6
	`, plan.Name, plan.Description, plan.Price, plan.Currency.String(), plan.DurationDays, plan.ID); err != nil {
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
