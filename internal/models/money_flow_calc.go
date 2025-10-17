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

// GetMoneyFlowSummaryRequest represents filter query parameters
type GetMoneyFlowSummaryRequest struct {
	PaymentType                   string `query:"paymentType" example:"MF_EARN_DIVEST"`
	TransactionSourceCreationDate string `query:"transactionSourceCreationDate" example:"2025-10-17"`
	Status                        string `query:"status" example:"PENDING"`
	Limit                         int    `query:"limit" example:"10"`
	NextCursor                    string `query:"nextCursor" example:"2"`
	PrevCursor                    string `query:"prevCursor" example:"1"`
}

// MoneyFlowSummaryResponse represents the response for money flow summary detail
type MoneyFlowSummaryResponse struct {
	Kind                          string          `json:"kind" example:"moneyFlowCalc"`
	ID                            string          `json:"id" example:"a232dd33-a036-44c7-8de5-0e8268f23267"`
	TransactionSourceCreationDate string          `json:"transactionSourceCreationDate" example:"2025-10-17"`
	PaymentType                   string          `json:"paymentType" example:"MF_EARN_DIVEST"`
	TotalTransfer                 decimal.Decimal `json:"totalTransfer" example:"24000"`
	Status                        string          `json:"status" example:"PENDING"`
	CreatedAt                     string          `json:"createdAt" example:"2025-10-16T22:18:29Z"`
	RequestedDate                 string          `json:"requestedDate" example:"-"`
	ActualDate                    string          `json:"actualDate" example:"-"`
}

// GetMoneyFlowSummaryListResponse represents the list response
type GetMoneyFlowSummaryListResponse struct {
	Kind     string                     `json:"kind" example:"collection"`
	Contents []MoneyFlowSummaryResponse `json:"contents"`
	//Pagination common.CursorPagination    `json:"pagination"`
}

// MoneyFlowSummaryOut represents the output from repository
type MoneyFlowSummaryOut struct {
	ID                            string
	TransactionSourceCreationDate time.Time
	PaymentType                   string
	TotalTransfer                 decimal.Decimal
	MoneyFlowStatus               string
	RequestedDate                 *time.Time
	ActualDate                    *time.Time
	CreatedAt                     time.Time
}

// ToModelResponse implements PaginateableContent interface
func (m MoneyFlowSummaryOut) ToModelResponse() MoneyFlowSummaryResponse {
	requestedDate := "-"
	if m.RequestedDate != nil {
		requestedDate = m.RequestedDate.Format(time.RFC3339)
	}

	actualDate := "-"
	if m.ActualDate != nil {
		actualDate = m.ActualDate.Format(time.RFC3339)
	}

	return MoneyFlowSummaryResponse{
		Kind:                          "moneyFlowCalc",
		ID:                            m.ID,
		TransactionSourceCreationDate: m.TransactionSourceCreationDate.Format("2006-01-02"),
		PaymentType:                   m.PaymentType,
		TotalTransfer:                 m.TotalTransfer,
		Status:                        m.MoneyFlowStatus,
		CreatedAt:                     m.CreatedAt.Format(time.RFC3339),
		RequestedDate:                 requestedDate,
		ActualDate:                    actualDate,
	}
}

// MoneyFlowSummaryFilterOptions represents filter options for database query
type MoneyFlowSummaryFilterOptions struct {
	PaymentType                   string
	TransactionSourceCreationDate *time.Time
	Status                        string
	Limit                         int
	Cursor                        *MoneFlowSummaryCursor
}

// MoneFlowSummaryCursor represents cursor for pagination
type MoneFlowSummaryCursor struct {
	ID         string
	IsBackward bool
}

type DoGetSummaryIDBySummaryIDRequest struct {
	SummaryID string `params:"summaryID" example:"bbc15647-0e2e-4f3a-9b2b-a4a918d3f34b"`
}

// MoneyFlowSummaryBySummaryIDOut represents the output from repository
type MoneyFlowSummaryBySummaryIDOut struct {
	Kind                         string
	ID                           string
	PaymentType                  string
	CreatedDate                  string
	RequestedDate                string
	ActualDate                   string
	TotalAmount                  decimal.Decimal
	Status                       string
	SourceBankAccountNumber      string
	DestinationBankAccountNumber string
}

type MoneyFlowSummaryDetailBySummaryIDOut struct {
	Kind                         string
	ID                           string
	PaymentType                  string
	CreatedDate                  time.Time
	RequestedDate                *time.Time
	ActualDate                   *time.Time
	TotalAmount                  decimal.Decimal
	Status                       string
	SourceBankAccountNumber      string
	DestinationBankAccountNumber string
}

func (m MoneyFlowSummaryDetailBySummaryIDOut) ToModelResponse() MoneyFlowSummaryBySummaryIDOut {
	requestedDate := "-"
	if m.RequestedDate != nil {
		requestedDate = m.RequestedDate.Format(time.RFC3339)
	}

	actualDate := "-"
	if m.ActualDate != nil {
		actualDate = m.ActualDate.Format(time.RFC3339)
	}
	return MoneyFlowSummaryBySummaryIDOut{
		Kind:                         "moneyFlowCalc",
		ID:                           m.ID,
		PaymentType:                  m.PaymentType,
		CreatedDate:                  m.CreatedDate.Format(time.RFC3339),
		RequestedDate:                requestedDate,
		ActualDate:                   actualDate,
		TotalAmount:                  m.TotalAmount,
		Status:                       m.Status,
		SourceBankAccountNumber:      m.SourceBankAccountNumber,
		DestinationBankAccountNumber: m.DestinationBankAccountNumber,
	}
}
