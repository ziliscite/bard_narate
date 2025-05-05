package domain

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"math"
	"time"
)

type PaymentStatus int

const (
	Pending PaymentStatus = iota
	Completed
	Failed
)

func (p PaymentStatus) String() string {
	return [...]string{"PENDING", "COMPLETED", "FAILED"}[p]
}

func (p PaymentStatus) EnumIndex() int {
	return int(p)
}

func ParsePaymentStatus(s string) (PaymentStatus, error) {
	switch s {
	case "PENDING":
		return Pending, nil
	case "COMPLETED":
		return Completed, nil
	case "FAILED":
		return Failed, nil
	default:
		return Pending, fmt.Errorf("invalid payment status: %s", s)
	}
}

type Order struct {
	ID        uuid.UUID
	UserID    uint64
	PlanID    uint64
	CreatedAt time.Time
}

func NewOrder(userID uint64, planID uint64) (*Order, error) {
	if userID == 0 {
		return nil, errors.New("user ID cannot be empty")
	}

	if planID == 0 {
		return nil, errors.New("plan ID cannot be empty")
	}

	return &Order{
		ID:        uuid.New(),
		UserID:    userID,
		PlanID:    planID,
		CreatedAt: time.Now(),
	}, nil
}

type Transaction struct {
	ID             uuid.UUID
	OrderID        uuid.UUID
	IdempotencyKey uuid.UUID // for payment gateway -- so that we don't double charge
	SubTotal       float64
	Tax            *float64
	ProcessingFee  *float64
	Discount       *float64
	Total          float64
	Currency       Currency
	Status         PaymentStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Version        int
}

func NewTransaction(
	orderId uuid.UUID,
	subTotal float64,
	currency string,
) (*Transaction, error) {
	if orderId == uuid.Nil {
		return nil, errors.New("order ID cannot be empty")
	}

	if subTotal <= 0 {
		return nil, errors.New("total must be greater than zero")
	}

	cur, err := NewCurrency(currency)
	if err != nil {
		return nil, fmt.Errorf("invalid currency: %w", err)
	}

	return &Transaction{
		ID:             uuid.New(),
		IdempotencyKey: uuid.New(),
		OrderID:        orderId,
		SubTotal:       subTotal,
		Currency:       cur,
		Status:         Pending,
	}, nil
}

func (t *Transaction) IsCompleted() bool {
	return t.Status == Completed
}

func (t *Transaction) Complete() {
	t.Status = Completed
}

func (t *Transaction) Cancel() {
	t.Status = Failed
}

type ProcessingOption func(*Transaction)

// WithDiscount adds discount in percentage. E.g., 10 -> 10% discount.
func WithDiscount(f float64) ProcessingOption {
	return func(s *Transaction) {
		s.Discount = &f
	}
}

// WithTax adds discount in percentage. E.g., 10 -> 10% discount.
func WithTax(f float64) ProcessingOption {
	return func(s *Transaction) {
		s.Tax = &f
	}
}

// WithFees adds discount in percentage. E.g., 10 -> 10% discount.
func WithFees(f float64) ProcessingOption {
	return func(s *Transaction) {
		s.ProcessingFee = &f
	}
}

// CalculateFinalAmount will calculate the final price for the transaction.
// All fees must be in percentage value. E.g., 10 for 10%.
func (t *Transaction) CalculateFinalAmount() error {
	if t.Discount != nil && (*t.Discount < 0 || *t.Discount > 100) {
		return errors.New("discount percentage must be between 0-100")
	}

	// Calculate discount amount
	discountAmount := 0.0
	if t.Discount != nil {
		discountAmount = t.SubTotal * (*t.Discount / 100)
	}

	discountedSubtotal := t.SubTotal - discountAmount

	if t.Tax != nil && *t.Tax < 0 {
		return errors.New("tax and fee percentages cannot be negative")
	}

	// Calculate tax on a discounted amount
	taxAmount := 0.0
	if t.Tax != nil {
		taxAmount = discountedSubtotal * (*t.Tax / 100)
	}

	// Calculate fee on (discounted + tax)
	feeAmount := 0.0
	if t.ProcessingFee != nil && *t.ProcessingFee < 0 {
		feeAmount = (discountedSubtotal + taxAmount) * (*t.ProcessingFee / 100)
	}

	// Sum the final total
	finalTotal := discountedSubtotal + taxAmount + feeAmount

	// Round to 2 decimal places for currency
	t.Total = math.Round(finalTotal*100) / 100
	if t.Total < 0.0 {
		return errors.New("total cannot be lower than 0")
	}

	return nil
}
