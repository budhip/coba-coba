package models

import "time"

// MoneyFlowSummary represents the money_flow_summaries table
type MoneyFlowSummary struct {
	ID              uint64    `db:"id"`
	TransactionType string    `db:"transaction_type"`
	TransactionDate time.Time `db:"transaction_date"`
	TotalTransfer   float64   `db:"total_transfer"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// DetailedMoneyFlowSummary represents the detailed_money_flow_summaries table
type DetailedMoneyFlowSummary struct {
	ID              uint64    `db:"id"`
	SummaryID       uint64    `db:"summary_id"`
	TransactionID   string    `db:"transaction_id"`
	RefNumber       string    `db:"ref_number"`
	Amount          float64   `db:"amount"`
	TransactionTime time.Time `db:"transaction_time"`
	CreatedAt       time.Time `db:"created_at"`
}

// CreateMoneyFlowSummary represents input for creating money flow summary
type CreateMoneyFlowSummary struct {
	TransactionType string
	TransactionDate time.Time
	TotalTransfer   float64
}

// UpdateMoneyFlowSummary represents input for updating money flow summary
type UpdateMoneyFlowSummary struct {
	ID              uint64
	TransactionType string
	TransactionDate time.Time
	TotalTransfer   float64
}

// CreateDetailedMoneyFlowSummary represents input for creating detailed money flow summary
type CreateDetailedMoneyFlowSummary struct {
	SummaryID       uint64
	TransactionID   string
	RefNumber       string
	Amount          float64
	TransactionTime time.Time
}
