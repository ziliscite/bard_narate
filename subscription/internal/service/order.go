package service

import (
	"context"
	"fmt"
	"github.com/ziliscite/bard_narate/subscription/internal/domain"
	"github.com/ziliscite/bard_narate/subscription/internal/repository"
)

type Order interface {
	Checkout(ctx context.Context, userID uint64, plan *domain.Plan, options ...domain.ProcessingOption) (*domain.Transaction, error)
	Finalize(ctx context.Context, transactionID string) error
	Cancel(ctx context.Context, transactionID string) error
}

type orderService struct {
	pr repository.PlanReader
	tr repository.Transaction
	sr repository.Subscription
}

func NewOrderService(pr repository.Plan, tr repository.Transaction, sr repository.Subscription) Order {
	return &orderService{
		pr: pr,
		tr: tr,
		sr: sr,
	}
}

func (o *orderService) Checkout(ctx context.Context, userID uint64, plan *domain.Plan, options ...domain.ProcessingOption) (*domain.Transaction, error) {
	// Create a new transaction
	transaction, err := o.tr.New(ctx, userID, plan.ID, plan.Currency.String(), plan.Price, func(tr *domain.Transaction) error {
		for _, opts := range options {
			opts(tr)
		}

		return tr.CalculateFinalAmount()
	})
	if err != nil {
		return nil, err
	}

	return transaction, nil
}

func (o *orderService) Finalize(ctx context.Context, transactionID string) error {
	transaction, order, err := o.tr.GetTransactionAndOrder(ctx, transactionID)
	if err != nil {
		return err
	}

	plan, err := o.pr.Get(ctx, order.PlanID)
	if err != nil {
		return err
	}

	activeSub, err := o.sr.GetActive(ctx, order.UserID)
	if err != nil {
		return err
	}

	newSub, err := domain.NewSubscription(order.UserID, plan.ID)
	if err != nil {
		return err
	}

	transaction.Complete()
	if activeSub != nil && !activeSub.IsExpired() {
		if activeSub.UserID != newSub.UserID {
			return fmt.Errorf("cannot transfer subscription between users")
		}

		return o.sr.PauseAndCreate(ctx, activeSub, newSub, transaction)
	}

	if activeSub != nil && activeSub.IsExpired() {
		activeSub.Deactivate()
		if err = o.sr.Update(ctx, activeSub.ID, activeSub); err != nil {
			return err
		}
	}

	// either there is no active sub or if it is expired, create a new sub
	return o.sr.Create(ctx, newSub, transaction)
}

func (o *orderService) Cancel(ctx context.Context, transactionID string) error {
	transaction, err := o.tr.GetTransaction(ctx, transactionID)
	if err != nil {
		return err
	}

	transaction.Cancel()
	return o.tr.Update(ctx, transaction.ID, transaction)
}
