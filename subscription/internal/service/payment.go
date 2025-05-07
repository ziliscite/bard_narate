package service

import (
	"context"
	"fmt"
	"github.com/ziliscite/bard_narate/subscription/internal/domain"
	"github.com/ziliscite/bard_narate/subscription/pkg/midtrans"
	"math"
)

type Payment interface {
	GetSnapURL(ctx context.Context, transaction *domain.Transaction, plan *domain.Plan) (string, error)
	VerifyPayment(ctx context.Context, notification *midtrans.PaymentStatus) (domain.PaymentStatus, error)
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
	// Round to make it an int
	total := int64(math.Ceil(transaction.Total))

	// Build request
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

func (p *paymentService) VerifyPayment(ctx context.Context, notification *midtrans.PaymentStatus) (domain.PaymentStatus, error) {
	//var notification midtrans.PaymentStatus
	//if err := json.Unmarshal(payload, &notification); err != nil {
	//	return 0, err
	//}

	// validate signature
	ok := p.pg.ValidateSignature(notification.OrderId, notification.StatusCode, notification.GrossAmount, notification.SignatureKey)
	if !ok {
		return 0, fmt.Errorf("invalid signature key")
	}

	// validate through status
	status, err := p.pg.GetTransactionStatus(ctx, notification.OrderId)
	if err != nil {
		return 0, err
	}

	if err = p.validatePayload(*notification, *status); err != nil {
		return 0, err
	}

	switch notification.TransactionStatus {
	case "capture":
		if notification.FraudStatus == "accept" {
			return domain.Completed, nil
		} else {
			return domain.Failed, nil
		}
	case "settlement":
		return domain.Completed, nil
	case "cancel", "deny", "expire", "failure":
		return domain.Failed, nil
	case "pending":
		return domain.Pending, nil
	default:
		return 0, fmt.Errorf("unknown transaction status: %s", notification.TransactionStatus)
	}
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

	if notification.StatusMessage != status.StatusMessage {
		return fmt.Errorf("invalid status message")
	}

	if notification.StatusCode != status.StatusCode {
		return fmt.Errorf("invalid status code")
	}

	if notification.SignatureKey != status.SignatureKey {
		return fmt.Errorf("invalid signature key")
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
