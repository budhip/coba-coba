package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

const IDRCurrency = "IDR"

type Amount struct {
	ValueDecimal Decimal `json:"value" validate:"required"`
	Currency     string  `json:"currency"`
}

// Scan implements the sql.Scanner interface
func (a *Amount) Scan(src interface{}) error {
	return a.ValueDecimal.Scan(src)
}

// Value implements the driver.Valuer interface
func (a Amount) Value() (value driver.Value, err error) {
	return a.ValueDecimal.Value()
}

type AmountDetail struct {
	Type   string  `json:"type" validate:"required"`
	Amount *Amount `json:"amount" validate:"required"`
}

type Amounts []AmountDetail

// Scan implements the sql.Scanner interface
func (b *Amounts) Scan(src interface{}) error {
	var raw []byte
	switch src := src.(type) {
	case string:
		raw = []byte(src)
	case []byte:
		raw = src
	default:
		return fmt.Errorf("type %T not supported by Scan", src)
	}

	return json.Unmarshal(raw, b)
}

// Value implements the driver.Valuer interface
func (b Amounts) Value() (value driver.Value, err error) {
	return json.Marshal(b)
}
