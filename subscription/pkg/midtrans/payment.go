package midtrans

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"resty.dev/v3"
)

// TransactionDetails represent transaction details.
//
// OrderID is a unique transaction ID. A single ID could be used only once by a Merchant.
// Allowed Symbols are dash(-), underscore(_), tilde (~), and dot (.).
// String(50) for Snap Checkout.
// String (36) for Payment Link.
//
// GrossAmt is the amount to be charged.
type TransactionDetails struct {
	OrderID  string `json:"order_id"`     // (required)
	GrossAmt int64  `json:"gross_amount"` // (required)
}

// ItemDetails represent the transaction details
type ItemDetails struct {
	ID           string `json:"id,omitempty"`            // (optional)
	Name         string `json:"name"`                    // (required)
	Price        int64  `json:"price"`                   // (required)
	Qty          int32  `json:"quantity"`                // (required)
	Brand        string `json:"brand,omitempty"`         // (optional)
	Category     string `json:"category,omitempty"`      // (optional)
	MerchantName string `json:"merchant_name,omitempty"` // (optional)
}

// CustomerDetails Represent the customer detail
type CustomerDetails struct {
	FName string `json:"first_name,omitempty"` // (required)
	LName string `json:"last_name,omitempty"`  // (optional)
	Email string `json:"email,omitempty"`      // (required)
	Phone string `json:"phone,omitempty"`      // (required)
}

// Response represent the midtrans payment response
type Response struct {
	Token         string   `json:"token"`
	RedirectURL   string   `json:"redirect_url"`
	StatusCode    string   `json:"status_code,omitempty"`
	ErrorMessages []string `json:"error_messages,omitempty"`
}

type SnapPaymentType string

const (
	// PaymentTypeCreditCard : credit_card
	PaymentTypeCreditCard SnapPaymentType = "credit_card"

	// PaymentTypeMandiriClickpay : mandiri_clickpay
	PaymentTypeMandiriClickpay SnapPaymentType = "mandiri_clickpay"

	// PaymentTypeCimbClicks : cimb_clicks
	PaymentTypeCimbClicks SnapPaymentType = "cimb_clicks"

	// PaymentTypeKlikBca : bca_klikbca
	PaymentTypeKlikBca SnapPaymentType = "bca_klikbca"

	// PaymentTypeBCAKlikpay : bca_klikpay
	PaymentTypeBCAKlikpay SnapPaymentType = "bca_klikpay"

	// PaymentTypeBRIEpay : bri_epay
	PaymentTypeBRIEpay SnapPaymentType = "bri_epay"

	// PaymentTypeTelkomselCash : telkomsel_cash
	PaymentTypeTelkomselCash SnapPaymentType = "telkomsel_cash"

	// PaymentTypeEChannel : echannel
	PaymentTypeEChannel SnapPaymentType = "echannel"

	// PaymentTypeMandiriEcash : mandiri_ecash
	PaymentTypeMandiriEcash SnapPaymentType = "mandiri_ecash"

	// PaymentTypePermataVA : permata_va
	PaymentTypePermataVA SnapPaymentType = "permata_va"

	// PaymentTypeOtherVA : other_va If you want to use other_va, either permata_va or bni_va
	// because Midtrans handles other bank transfer as either Permata or BNI VA.
	PaymentTypeOtherVA SnapPaymentType = "other_va"

	// PaymentTypeBCAVA : bca_va
	PaymentTypeBCAVA SnapPaymentType = "bca_va"

	// PaymentTypeBNIVA : bni_va
	PaymentTypeBNIVA SnapPaymentType = "bni_va"

	// PaymentTypeBRIVA : bca_va
	PaymentTypeBRIVA SnapPaymentType = "bri_va"

	// PaymentTypeBankTransfer : bank_transfer
	PaymentTypeBankTransfer SnapPaymentType = "bank_transfer"

	// PaymentTypeConvenienceStore : cstore
	PaymentTypeConvenienceStore SnapPaymentType = "cstore"

	// PaymentTypeIndomaret : indomaret
	PaymentTypeIndomaret SnapPaymentType = "indomaret"

	// PaymentTypeKioson : kioson
	PaymentTypeKioson SnapPaymentType = "kioson"

	// PaymentTypeDanamonOnline : danamon_online
	PaymentTypeDanamonOnline SnapPaymentType = "danamon_online"

	// PaymentTypeAkulaku : akulaku
	PaymentTypeAkulaku SnapPaymentType = "akulaku"

	// PaymentTypeGopay : gopay
	PaymentTypeGopay SnapPaymentType = "gopay"

	// PaymentTypeShopeepay : shopeepay
	PaymentTypeShopeepay SnapPaymentType = "shopeepay"

	// PaymentTypeAlfamart : alfamart
	PaymentTypeAlfamart SnapPaymentType = "alfamart"
)

// AllSnapPaymentType Get All available SnapPaymentType
var AllSnapPaymentType = []SnapPaymentType{
	PaymentTypeGopay,
	PaymentTypeShopeepay,
	PaymentTypeCreditCard,
	PaymentTypeBankTransfer,
	PaymentTypeBNIVA,
	PaymentTypePermataVA,
	PaymentTypeBCAVA,
	PaymentTypeBRIVA,
	PaymentTypeOtherVA,
	PaymentTypeMandiriClickpay,
	PaymentTypeCimbClicks,
	PaymentTypeDanamonOnline,
	PaymentTypeKlikBca,
	PaymentTypeBCAKlikpay,
	PaymentTypeBRIEpay,
	PaymentTypeMandiriEcash,
	PaymentTypeTelkomselCash,
	PaymentTypeEChannel,
	PaymentTypeIndomaret,
	PaymentTypeKioson,
	PaymentTypeAkulaku,
	PaymentTypeAlfamart,
	PaymentTypeConvenienceStore,
}

// SnapTokenRequest represent the snap token request.
// https://docs.midtrans.com/reference/snap-api-overview
//
// TransactionDetails represent the transaction details. Required.
//
// Items represent the transaction details. Optional.
//
// CustomerDetails represent the customer details. Optional.
//
// EnabledPayments represent the enabled payments. Optional.
type SnapTokenRequest struct {
	TransactionDetails TransactionDetails `json:"transaction_details"`        // (required)
	Items              []ItemDetails      `json:"items,omitempty"`            // (optional)
	CustomerDetails    CustomerDetails    `json:"customer_details,omitempty"` // (optional)
	EnabledPayments    []SnapPaymentType  `json:"enabled_payments,omitempty"` // (optional)
}

func (c *Client) GetSnapToken(ctx context.Context, request SnapTokenRequest, idempotencyKey uuid.UUID) (*Response, error) {
	res := &Response{}

	resp, err := c.snapRequest(ctx, request).
		SetResult(res).
		SetHeader("Idempotency-Key", idempotencyKey.String()).
		Post("/snap/v1/transactions")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, errors.New("failed to get snap token")
	}

	return res, nil
}

func (c *Client) snapRequest(ctx context.Context, sr SnapTokenRequest) *resty.Request {
	return c.requestJSON(ctx).SetBody(sr)
}
