package domain

import (
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"
)

type SubscriptionStatus int

const (
	Active SubscriptionStatus = iota
	Paused
	Expired
)

func (p SubscriptionStatus) String() string {
	return [...]string{"ACTIVE", "PAUSED", "EXPIRED"}[p]
}

func (p SubscriptionStatus) EnumIndex() int {
	return int(p)
}

func ParseSubscriptionStatus(s string) (SubscriptionStatus, error) {
	switch strings.ToUpper(s) {
	case "ACTIVE":
		return Active, nil
	case "PAUSED":
		return Paused, nil
	case "EXPIRED":
		return Expired, nil
	default:
		return Active, fmt.Errorf("invalid payment status: %s", s)
	}
}

type Subscription struct {
	ID            uuid.UUID
	UserID        uint64
	PlanID        uint64
	StartDate     time.Time
	EndDate       time.Time
	Status        SubscriptionStatus
	PausedAt      time.Time
	RemainingDays int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Version       int
}

func NewSubscription(
	userID uint64,
	planID uint64,
) (*Subscription, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	if planID == 0 {
		return nil, fmt.Errorf("plan ID cannot be empty")
	}

	return &Subscription{
		ID:        uuid.New(),
		UserID:    userID,
		PlanID:    planID,
		StartDate: time.Now(),
		Status:    Active,
	}, nil
}

func (s *Subscription) IsExpired() bool {
	return time.Now().After(s.EndDate)
}

// Deactivate sets the subscription status to Expire.
// It should be called by a cron job when the subscription is no longer active.
// or when transaction has failed.
// (IDK, maybe use database transaction and dont insert it if failed?).
func (s *Subscription) Deactivate() {
	s.Status = Expired
}
