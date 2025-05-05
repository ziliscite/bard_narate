package midtrans

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
)

type PaymentStatus struct {
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"`
	TransactionId     string `json:"transaction_id"`
	StatusMessage     string `json:"status_message"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	OrderId           string `json:"order_id"`
	MerchantId        string `json:"merchant_id"`
	GrossAmount       string `json:"gross_amount"`
	FraudStatus       string `json:"fraud_status"`
	Currency          string `json:"currency"`
}

func (c *Client) GetTransactionStatus(ctx context.Context, orderID string) (*PaymentStatus, error) {
	res := &PaymentStatus{}

	resp, err := c.requestJSON(ctx).
		SetResult(res).
		SetPathParam("orderID", orderID).
		Get("/v2/{orderID}/status")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, errors.New("failed to get transaction status")
	}

	return res, nil
}

func (c *Client) ValidateSignature(orderId, statusCode, grossAmount, signatureKey string) bool {
	hash := sha512.Sum512([]byte(orderId + statusCode + grossAmount + c.sk))
	return hex.EncodeToString(hash[:]) == signatureKey
}
