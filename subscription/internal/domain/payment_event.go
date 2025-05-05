package domain

import (
	"errors"
	"github.com/google/uuid"
	"time"
)

type PaymentEvent struct {
	ID            uuid.UUID
	TransactionID uuid.UUID
	Payload       []byte
	ReceivedAt    time.Time
	Status        string
}

func NewPaymentEvent(
	transactionID uuid.UUID,
	payload []byte,
	status string,
) (*PaymentEvent, error) {
	if transactionID == uuid.Nil {
		return nil, errors.New("transaction ID cannot be empty")
	}

	if len(payload) == 0 {
		return nil, errors.New("payload cannot be empty")
	}

	if status == "" {
		return nil, errors.New("status cannot be empty")
	}

	return &PaymentEvent{
		ID:            uuid.New(),
		TransactionID: transactionID,
		Payload:       payload,
		Status:        status,
		ReceivedAt:    time.Now(),
	}, nil
}
