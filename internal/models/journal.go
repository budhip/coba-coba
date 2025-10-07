package models

import "github.com/shopspring/decimal"

type JournalStreamTransaction struct {
	TransactionType string          `json:"transactionType"`
	Account         string          `json:"account"`
	Narrative       string          `json:"narrative"`
	Amount          decimal.Decimal `json:"amount"`
	IsDebit         bool            `json:"isDebit"`
}

type JournalStreamPayload struct {
	Type            string                     `json:"type"`
	TransactionID   string                     `json:"transactionId"`
	OrderType       string                     `json:"orderType"`
	TransactionDate string                     `json:"transactionDate"`
	ProcessingDate  string                     `json:"processingDate"`
	Currency        string                     `json:"currency"`
	Transactions    []JournalStreamTransaction `json:"transactions"`
	Metadata        any                        `json:"metadata"`
}
