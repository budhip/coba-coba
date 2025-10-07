package common

import "github.com/shopspring/decimal"

// NewDecimalFromString converts a string to a decimal.Decimal pointer.
// It parses the input string and returns a pointer to the decimal value,
// along with any parsing error. If the input string is empty, it returns nil.
func NewDecimalFromString(data string) (*decimal.Decimal, error) {
	if data != "" {
		amount, err := decimal.NewFromString(data)
		if err != nil {
			return nil, err
		}
		return &amount, nil
	}
	return nil, nil
}
