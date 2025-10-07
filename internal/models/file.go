package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// CtxKeyNgmisHeader is the header key for the username of the user making the request.
// currently only used in ngmis manual upload transaction
const CtxKeyNgmisHeader = "X-Ngmis-Username"

type FileOut struct {
	Kind   string `json:"kind"`
	File   string `json:"file"`
	Status string `json:"status"`
}

func NewFileOut(file, status string) *FileOut {
	return &FileOut{
		Kind:   "file",
		Status: status,
		File:   file,
	}
}

type FileTransaction struct {
	TransactionDate      *time.Time
	OrderType            string
	Amount               *decimal.Decimal
	Currency             string
	SourceAccountId      string
	DestinationAccountId string
	Description          string
	Method               string
	TransactionType      string
}
