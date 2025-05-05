package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidDiscount  = errors.New("discount is not applicable to this plan")
	ErrExpiredDiscount  = errors.New("discount is expired")
	ErrInActiveDiscount = errors.New("discount is not yet active")
)

type Plan struct {
	ID           uint64
	Name         string
	Description  string
	Price        float64
	Currency     Currency
	DurationDays int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Version      int
}

func NewPlan(
	name, description string,
	price float64, currency string,
	durationDays int,
) (*Plan, error) {
	if name == "" {
		return nil, errors.New("plan name cannot be empty")
	}

	if description == "" {
		return nil, errors.New("plan description cannot be empty")
	}

	if price <= 0 {
		return nil, errors.New("plan price must be greater than zero")
	}

	if durationDays <= 0 {
		return nil, errors.New("plan duration must be greater than zero")
	}

	cur, err := NewCurrency(currency)
	if err != nil {
		return nil, err
	}

	if durationDays%30 != 0 && (durationDays < 30 || durationDays > 90) {
		return nil, errors.New("plan duration must be divisible by 30 and is between 30 and 90 days")
	}

	return &Plan{
		Name:         name,
		Description:  description,
		Price:        price,
		Currency:     cur,
		DurationDays: durationDays,
	}, nil
}

// EndDate calculates the end date of the plan based on the start date
func (p *Plan) EndDate(startDate time.Time) time.Time {
	return startDate.AddDate(0, 0, p.DurationDays)
}

// IsExpired checks if the plan is expired based on the start date
func (p *Plan) IsExpired(startDate time.Time) bool {
	return time.Now().After(p.EndDate(startDate))
}

// CalculatePriceWithTax calculates the price with tax
func (p *Plan) CalculatePriceWithTax(taxRate float64) float64 {
	return p.Price + (p.Price * taxRate)
}

// DiscountScope represents the scope of the discount
type DiscountScope int

const (
	AllScope DiscountScope = iota
	PlanScope
)

func (d DiscountScope) String() string {
	return [...]string{"ALL", "PLAN"}[d]
}

func (d DiscountScope) EnumIndex() int {
	return int(d)
}

func ParseDiscountScope(s string) (DiscountScope, error) {
	switch s {
	case "ALL":
		return AllScope, nil
	case "PLAN":
		return PlanScope, nil
	default:
		return AllScope, errors.New("invalid discount scope")
	}
}

// Discount represents a discount applied to a plan
type Discount struct {
	ID              uint64
	Code            string
	Description     string
	Scope           DiscountScope
	PercentageValue float64
	StartDate       time.Time
	EndDate         time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Version         int
}

func NewDiscount(
	code, description string,
	scope string,
	value float64,
	startDate, endDate time.Time,
) (*Discount, error) {
	if code == "" {
		return nil, errors.New("discount code cannot be empty")
	}

	if description == "" {
		return nil, errors.New("discount description cannot be empty")
	}

	scopeEnum, err := ParseDiscountScope(scope)
	if err != nil {
		return nil, err
	}

	if value <= 0 {
		return nil, errors.New("discount value must be greater than zero")
	}

	// there might be a case for 100% discount
	if value > 100 {
		return nil, errors.New("discount value must be less or equal to 100")
	}

	if startDate.IsZero() {
		return nil, errors.New("discount start date cannot be empty")
	}

	if endDate.IsZero() {
		return nil, errors.New("discount end date cannot be empty")
	}

	return &Discount{
		Code:            code,
		Description:     description,
		Scope:           scopeEnum,
		PercentageValue: value,
		StartDate:       startDate,
		EndDate:         endDate,
	}, nil
}

func (d *Discount) IsExpired() bool {
	return time.Now().After(d.EndDate)
}

func (d *Discount) IsActive() bool {
	return time.Now().After(d.StartDate)
}

func (d *Discount) NewPlanDiscount(plan *Plan) (*DiscountPlan, error) {
	if plan == nil {
		return nil, errors.New("plan cannot be nil")
	}

	if d.Scope != PlanScope {
		return nil, errors.New("discount scope is not applicable to this plan")
	}

	if d.IsExpired() {
		return nil, errors.New("discount is expired")
	}

	if !d.IsActive() {
		return nil, errors.New("discount is not yet active")
	}

	return &DiscountPlan{
		DiscountID: d.ID,
		PlanID:     plan.ID,
	}, nil
}

type DiscountPlan struct {
	DiscountID uint64
	PlanID     uint64
}

func NewDiscountPlan(
	discountId uint64,
	planId uint64,
) (*DiscountPlan, error) {
	if discountId == 0 {
		return nil, errors.New("discount ID cannot be empty")
	}

	if planId == 0 {
		return nil, errors.New("plan ID cannot be empty")
	}

	return &DiscountPlan{
		DiscountID: discountId,
		PlanID:     planId,
	}, nil
}
