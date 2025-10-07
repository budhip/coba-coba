package acuanclient

import (
	"time"

	goAcuanLibModel "bitbucket.org/Amartha/go-acuan-lib/model"
	"github.com/shopspring/decimal"
)

type PublishTransactionRequest struct {
	FromAccount     string                            `json:"fromAccount"`
	ToAccount       string                            `json:"toAccount"`
	Amount          decimal.Decimal                   `json:"amount"`
	Method          goAcuanLibModel.TransactionMethod `json:"method"`
	TransactionType goAcuanLibModel.TransactionType   `json:"transactionType"`
	TransactionTime time.Time                         `json:"transactionTime"`
	OrderType       string                            `json:"orderType"`
	RefNumber       string                            `json:"refNumber"`
	Description     string                            `json:"description"`
	Metadata        interface{}                       `json:"metadata"`
	Currency        string                            `json:"currency"`
}
