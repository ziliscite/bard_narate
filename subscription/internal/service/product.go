package service

import (
	"context"
	"github.com/ziliscite/bard_narate/subscription/internal/domain"
	"github.com/ziliscite/bard_narate/subscription/internal/repository"
)

type PlanReader interface {
	GetPlan(ctx context.Context, id uint64) (*domain.Plan, error)
	GetAllPlans(ctx context.Context) ([]*domain.Plan, error)
}

type PlanWriter interface {
	CreatePlan(ctx context.Context, plan *domain.Plan) error
	UpdatePlan(ctx context.Context, plan *domain.Plan) error
}

type Plan interface {
	PlanReader
	PlanWriter
}

type DiscountReader interface {
	GetDiscount(ctx context.Context, code string, planID uint64) (*domain.Discount, error)
}

type DiscountWriter interface {
	CreateDiscount(ctx context.Context, discount *domain.Discount) error
	UpdateDiscount(ctx context.Context, discount *domain.Discount) error
	AttachPlansToDiscount(ctx context.Context, discountID uint64, planIDS ...uint64) error
}

type Discount interface {
	DiscountReader
	DiscountWriter
}

type ProductReader interface {
	PlanReader
	DiscountReader
}

type ProductWriter interface {
	PlanWriter
	DiscountWriter
}

type Product interface {
	ProductReader
	ProductWriter
}

type productService struct {
	pr repository.Plan
	dr repository.Discount
}

func NewPlanService(pr repository.Plan, dr repository.Discount) Product {
	return &productService{
		pr: pr,
		dr: dr,
	}
}

func (ps *productService) GetAllPlans(ctx context.Context) ([]*domain.Plan, error) {
	return ps.pr.GetAll(ctx)
}

func (ps *productService) GetPlan(ctx context.Context, id uint64) (*domain.Plan, error) {
	return ps.pr.Get(ctx, id)
}

func (ps *productService) CreatePlan(ctx context.Context, plan *domain.Plan) error {
	return ps.pr.Create(ctx, plan)
}

func (ps *productService) UpdatePlan(ctx context.Context, plan *domain.Plan) error {
	return ps.pr.Update(ctx, plan)
}

func (ps *productService) GetDiscount(ctx context.Context, code string, planID uint64) (*domain.Discount, error) {
	if code == "" {
		return nil, domain.ErrEmptyDiscount
	}

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

func (ps *productService) CreateDiscount(ctx context.Context, discount *domain.Discount) error {
	return ps.dr.Create(ctx, discount)
}

func (ps *productService) UpdateDiscount(ctx context.Context, discount *domain.Discount) error {
	return ps.dr.Update(ctx, discount)
}

func (ps *productService) AttachPlansToDiscount(ctx context.Context, discountID uint64, planIDS ...uint64) error {
	return ps.dr.AttachPlansToDiscount(ctx, discountID, planIDS...)
}
