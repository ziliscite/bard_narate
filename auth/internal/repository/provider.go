package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/ziliscite/bard_narate/auth/internal/domain"
)

type Provider interface {
	Create(ctx context.Context, provider *domain.Provider) error
	GetAll(ctx context.Context) ([]*domain.Provider, error)
	Update(ctx context.Context, name string, provider *domain.Provider) error
}

type providerRepo struct {
	db *pgxpool.Pool
}

func NewProviderRepository(db *pgxpool.Pool) (Provider, error) {
	if db == nil {
		return nil, ErrNilPool
	}

	return &providerRepo{db: db}, nil
}

func (r *providerRepo) Create(ctx context.Context, provider *domain.Provider) error {
	if _, err := r.db.Exec(ctx, `
        INSERT INTO providers (name, enabled)
        VALUES ($1, $2)
    `, provider.Name, provider.Enabled); err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			return fmt.Errorf("provider already exists: %w", ErrDuplicate)
		default:
			return fmt.Errorf("%w: %w", ErrUnknown, err)
		}
	}

	return nil
}

func (r *providerRepo) GetAll(ctx context.Context) ([]*domain.Provider, error) {
	rows, err := r.db.Query(ctx, `
        SELECT name, enabled, created_at, updated_at
        FROM providers
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*domain.Provider
	for rows.Next() {
		var p domain.Provider
		if err := rows.Scan(&p.Name, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrUnknown, err)
		}

		providers = append(providers, &p)
	}

	if len(providers) == 0 {
		return nil, ErrNotFound
	}

	return providers, nil
}

func (r *providerRepo) Update(ctx context.Context, name string, provider *domain.Provider) error {
	if _, err := r.db.Exec(ctx, `
        UPDATE providers
        SET name = $1, enabled = $2
        WHERE name = $3
    `, provider.Name, provider.Enabled, name); err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("provider not found: %w", ErrNotFound)
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			return fmt.Errorf("provider already exists: %w", ErrDuplicate)
		default:
			return fmt.Errorf("%w: %w", ErrUnknown, err)
		}
	}

	return nil
}
