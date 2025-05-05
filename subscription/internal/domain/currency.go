package domain

import "errors"

type Currency struct {
	code string
	name string
}

var (
	IDR = Currency{
		code: "IDR",
		name: "Indonesian Rupiah",
	}
	USD = Currency{
		code: "USD",
		name: "United States Dollar",
	}
)

func NewCurrency(code string) (Currency, error) {
	if code == "" {
		return Currency{}, errors.New("currency code cannot be empty")
	}

	switch code {
	case IDR.code:
		return IDR, nil
	case USD.code:
		return USD, nil
	default:
		return Currency{}, errors.New("invalid currency code")
	}
}

func (c Currency) String() string {
	return c.code
}

func (c Currency) IsZero() bool {
	return c == Currency{}
}
