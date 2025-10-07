package models

import (
	"github.com/shopspring/decimal"
)

// Decimal is a custom type for decimal.Decimal
// the difference from `shopspring` is the json representation is without quotes
// for example the result of this type is 10 instead of "10"
//
// WARNING: if client side is using javascript and unmarshalling this type, the precision will be lost
// since javascript will unmarshal JSON numbers to IEEE 754 double-precision floating point numbers
type Decimal struct {
	decimal.Decimal
}

func NewDecimalFromExternal(d decimal.Decimal) Decimal {
	return Decimal{d}
}

func NewDecimal(value string) (Decimal, error) {
	d, err := decimal.NewFromString(value)
	if err != nil {
		return Decimal{}, err
	}

	return Decimal{d}, nil
}

func (d Decimal) MarshalJSON() ([]byte, error) {
	return []byte(d.String()), nil
}
