package models

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// MoneyFlowSummary represents the money_flow_summaries table
type MoneyFlowSummary struct {
	ID                            string          `db:"id"`
	TransactionSourceCreationDate time.Time       `db:"transaction_source_date"`
	TransactionType               string          `db:"transaction_type"`
	PaymentType                   string          `db:"payment_type"`
	ReferenceNumber               string          `db:"reference_number"`
	Description                   string          `db:"description"`
	SourceAccount                 string          `db:"source_account"`
	DestinationAccount            string          `db:"destination_account"`
	TotalTransfer                 decimal.Decimal `db:"total_transfer"`
	PapaTransactionID             string          `db:"papa_transaction_id"`
	MoneyFlowStatus               string          `db:"money_flow_status"`
	RequestedDate                 *time.Time      `db:"requested_date"`
	ActualDate                    *time.Time      `db:"actual_date"`
	SourceBankAccountNumber       string          `db:"source_bank_account_number"`
	SourceBankAccountName         string          `db:"source_bank_account_name"`
	SourceBankName                string          `db:"source_bank_name"`
	DestinationBankAccountNumber  string          `db:"destination_bank_account_number"`
	DestinationBankAccountName    string          `db:"destination_bank_account_name"`
	DestinationBankName           string          `db:"destination_bank_name"`
	CreatedAt                     time.Time       `db:"created_at"`
	UpdatedAt                     time.Time       `db:"updated_at"`
}

// DetailedMoneyFlowSummary represents the detailed_money_flow_summaries table
type DetailedMoneyFlowSummary struct {
	ID                 string    `db:"id"`
	SummaryID          string    `db:"summary_id"`
	AcuanTransactionID string    `db:"acuan_transaction_id"`
	CreatedAt          time.Time `db:"created_at"`
	UpdatedAt          time.Time `db:"updated_at"`
}

// CreateMoneyFlowSummary represents input for creating money flow summary
type CreateMoneyFlowSummary struct {
	ID                            string
	TransactionSourceCreationDate time.Time
	TransactionType               string
	PaymentType                   string
	ReferenceNumber               string
	Description                   string
	SourceAccount                 string
	DestinationAccount            string
	TotalTransfer                 float64
	PapaTransactionID             string
	MoneyFlowStatus               string
	RequestedDate                 *time.Time
	ActualDate                    *time.Time
	SourceBankAccountNumber       string
	SourceBankAccountName         string
	SourceBankName                string
	DestinationBankAccountNumber  string
	DestinationBankAccountName    string
	DestinationBankName           string
}

// MoneyFlowTransactionProcessed represents the transaction still processed in money_flow_summaries table
type MoneyFlowTransactionProcessed struct {
	ID                            string          `db:"id"`
	TransactionSourceCreationDate time.Time       `db:"transaction_source_date"`
	TransactionType               string          `db:"transaction_type"`
	PaymentType                   string          `db:"payment_type"`
	TotalTransfer                 decimal.Decimal `db:"total_transfer"`
	MoneyFlowStatus               string          `db:"money_flow_status"`
}

// CreateDetailedMoneyFlowSummary represents input for creating detailed money flow summary
type CreateDetailedMoneyFlowSummary struct {
	ID                 string
	SummaryID          string
	AcuanTransactionID string
}

type TransactionNotificationRaw struct {
	AcuanData json.RawMessage `json:"acuanData"`
}

// MoneyFlowSummaryUpdate struct for update with optional fields
// Use pointers to distinguish between zero value and not set
type MoneyFlowSummaryUpdate struct {
	PaymentType       *string          `json:"payment_type,omitempty"`
	TotalTransfer     *decimal.Decimal `json:"total_transfer,omitempty"`
	PapaTransactionID *string          `json:"papa_transaction_id,omitempty"`
	MoneyFlowStatus   *string          `json:"money_flow_status,omitempty"`
	RequestedDate     *time.Time       `json:"requested_date,omitempty"`
	ActualDate        *time.Time       `json:"actual_date,omitempty"`
}

type BusinessRulesConfigs struct {
	BusinessRulesConfigs    map[string]BusinessRuleConfig `json:"payment_configs"`
	TransactionToPaymentMap map[string]string             `json:"transaction_to_payment_map"`
}

type BusinessRuleConfig struct {
	TransactionType string        `json:"transaction_type"`
	RequestToPAPA   RequestToPAPA `json:"request_to_papa"`
	Source          BankInfo      `json:"source"`
	Destination     BankInfo      `json:"destination"`
}

type RequestToPAPA struct {
	Description string `json:"description"`
}

type BankInfo struct {
	AccountNumber     string `json:"account_number"`
	BankCode          string `json:"bank_code"`
	BankName          string `json:"bank_name"`
	BankAccountNumber string `json:"bank_account_number"`
	BankAccountName   string `json:"bank_account_name"`
}
