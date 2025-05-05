package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ziliscite/bard_narate/subscription/internal/domain"
	"github.com/ziliscite/bard_narate/subscription/pkg/midtrans"
	"math"
)

type Payment interface {
	GetSnapURL(ctx context.Context, transaction *domain.Transaction, plan *domain.Plan) (string, error)
	VerifyPayment(ctx context.Context, payload []byte) (error, domain.PaymentStatus)
}

type paymentService struct {
	pg *midtrans.Client
}

func NewPayment(pg *midtrans.Client) Payment {
	return &paymentService{
		pg: pg,
	}
}

func (p *paymentService) GetSnapURL(ctx context.Context, transaction *domain.Transaction, plan *domain.Plan) (string, error) {
	total := int64(math.Ceil(transaction.Total))
	request := midtrans.SnapTokenRequest{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  transaction.ID.String(),
			GrossAmt: total,
		},
		Items: []midtrans.ItemDetails{
			{
				ID:           fmt.Sprintf("%d", plan.ID),
				Name:         plan.Name,
				Price:        total,
				Qty:          1,
				Brand:        "Bard Narate",
				Category:     "Services",
				MerchantName: "Ziliscite",
			},
		},
	}

	// Call payment gateway to process the payment
	res, err := p.pg.GetSnapToken(ctx, request, transaction.IdempotencyKey)
	if err != nil {
		return "", err
	}

	return res.RedirectURL, nil
}

func (p *paymentService) VerifyPayment(ctx context.Context, payload []byte) (error, domain.PaymentStatus) {
	var notification midtrans.PaymentStatus
	if err := json.Unmarshal(payload, &notification); err != nil {
		return err, 0
	}

	// validate signature
	ok := p.pg.ValidateSignature(notification.OrderId, notification.StatusCode, notification.GrossAmount, notification.SignatureKey)
	if !ok {
		return fmt.Errorf("invalid signature key"), 0
	}

	// validate through status
	status, err := p.pg.GetTransactionStatus(ctx, notification.OrderId)
	if err != nil {
		return err, 0
	}

	if err = p.validatePayload(notification, *status); err != nil {
		return err, 0
	}

	switch notification.TransactionStatus {
	case "capture":
		if notification.FraudStatus == "accept" {
			return nil, domain.Completed
		}
	case "settlement":
		return nil, domain.Completed
	case "cancel", "deny", "expire", "failure":
		return nil, domain.Failed
	case "pending":
		return nil, domain.Pending
	default:
		return fmt.Errorf("unknown transaction status: %s", notification.TransactionStatus), 0
	}

	return fmt.Errorf("unknown transaction status: %s", notification.TransactionStatus), 0
}

func (p *paymentService) validatePayload(notification, status midtrans.PaymentStatus) error {
	if notification.TransactionTime != status.TransactionTime {
		return fmt.Errorf("invalid transaction time")
	}

	if notification.TransactionStatus != status.TransactionStatus {
		return fmt.Errorf("invalid transaction status")
	}

	if notification.TransactionId != status.TransactionId {
		return fmt.Errorf("invalid transaction id")
	}

	if notification.OrderId != status.OrderId {
		return fmt.Errorf("invalid status code")
	}

	if notification.GrossAmount != status.GrossAmount {
		return fmt.Errorf("invalid gross amount")
	}

	if notification.FraudStatus != status.FraudStatus {
		return fmt.Errorf("invalid fraud status")
	}

	if notification.Currency != status.Currency {
		return fmt.Errorf("invalid currency")
	}

	return nil
}
