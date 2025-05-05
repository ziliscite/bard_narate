package repository

import (
	"context"
	"github.com/ziliscite/bard_narate/subscription/internal/domain"
)

type Plan interface {
	// Get retrieves a plan by its ID from the database.
	Get(ctx context.Context, id uint64) (*domain.Plan, error)
	// GetAll retrieves all plans from the database.
	GetAll(ctx context.Context) ([]*domain.Plan, error)
	// Create creates a new plan in the database.
	Create(ctx context.Context, plan *domain.Plan) error
}
