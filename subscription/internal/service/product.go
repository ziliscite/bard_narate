package service

import (
	"context"
	"github.com/ziliscite/bard_narate/subscription/internal/domain"
	"github.com/ziliscite/bard_narate/subscription/internal/repository"
)

type Plan interface {
	GetPlan(ctx context.Context, id uint64) (*domain.Plan, error)
	GetDiscount(ctx context.Context, code string, planID uint64) (*domain.Discount, error)
}

type planService struct {
	pr repository.Plan
	dr repository.Validator
}

func NewPlanService(pr repository.Plan, dr repository.Discount) Plan {
	return &planService{
		pr: pr,
		dr: dr,
	}
}

func (ps planService) GetPlan(ctx context.Context, id uint64) (*domain.Plan, error) {
	return ps.pr.Get(ctx, id)
}

func (ps planService) GetDiscount(ctx context.Context, code string, planID uint64) (*domain.Discount, error) {
	discount, ok, err := ps.dr.ValidateAndGet(ctx, code, planID)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, domain.ErrInvalidDiscount
	}

	if discount.IsExpired() {
		return nil, domain.ErrExpiredDiscount
	}

	if !discount.IsActive() {
		return nil, domain.ErrInActiveDiscount
	}

	return discount, nil
}
