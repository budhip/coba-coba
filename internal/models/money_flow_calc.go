package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/constants"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/dateutil"

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
	PaymentType                        string `query:"paymentType" example:"MF_EARN_DIVEST"`
	TransactionSourceCreationDateStart string `query:"transactionSourceCreationDateStart" example:"2025-10-17"`
	TransactionSourceCreationDateEnd   string `query:"transactionSourceCreationDateEnd" example:"2025-10-20"`
	Status                             string `query:"status" example:"PENDING"`
	Limit                              int    `query:"limit" example:"10"`
	NextCursor                         string `query:"nextCursor" example:"2"`
	PrevCursor                         string `query:"prevCursor" example:"1"`
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
	dates := dateutil.FormatTimesToRFC3339(m.RequestedDate, m.ActualDate)

	return MoneyFlowSummaryResponse{
		Kind:                          constants.MoneyFlowKind,
		ID:                            m.ID,
		TransactionSourceCreationDate: m.TransactionSourceCreationDate.Format(constants.DateFormatYYYYMMDD),
		PaymentType:                   m.PaymentType,
		TotalTransfer:                 m.TotalTransfer,
		Status:                        m.MoneyFlowStatus,
		CreatedAt:                     m.CreatedAt.Format(time.RFC3339),
		RequestedDate:                 dates[0],
		ActualDate:                    dates[1],
	}
}

// MoneyFlowSummaryFilterOptions represents filter options for database query
type MoneyFlowSummaryFilterOptions struct {
	PaymentType                        string
	TransactionSourceCreationDateStart *time.Time
	TransactionSourceCreationDateEnd   *time.Time
	Status                             string
	Limit                              int
	Cursor                             *MoneyFlowSummaryCursor
}

// MoneFlowSummaryCursor represents cursor for pagination
type MoneyFlowSummaryCursor struct {
	ID         string
	IsBackward bool
}

func (c MoneyFlowSummaryCursor) String() string {
	return c.ID
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
	SourceBankAccountName        string
	SourceBankName               string
	DestinationBankAccountNumber string
	DestinationBankAccountName   string
	DestinationBankName          string
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
	SourceBankAccountName        string
	SourceBankName               string
	DestinationBankAccountNumber string
	DestinationBankAccountName   string
	DestinationBankName          string
}

func (m MoneyFlowSummaryDetailBySummaryIDOut) ToModelResponse() MoneyFlowSummaryBySummaryIDOut {
	dates := dateutil.FormatTimesToRFC3339(m.RequestedDate, m.ActualDate)

	return MoneyFlowSummaryBySummaryIDOut{
		Kind:                         constants.MoneyFlowKind,
		ID:                           m.ID,
		PaymentType:                  m.PaymentType,
		CreatedDate:                  m.CreatedDate.Format(time.RFC3339),
		RequestedDate:                dates[0],
		ActualDate:                   dates[1],
		TotalAmount:                  m.TotalAmount,
		Status:                       m.Status,
		SourceBankAccountNumber:      m.SourceBankAccountNumber,
		SourceBankAccountName:        m.SourceBankAccountName,
		SourceBankName:               m.SourceBankName,
		DestinationBankAccountNumber: m.DestinationBankAccountNumber,
		DestinationBankAccountName:   m.DestinationBankAccountName,
		DestinationBankName:          m.DestinationBankName,
	}
}

// DoGetDetailedTransactionsBySummaryIDRequest represents request to get detailed transactions
type DoGetDetailedTransactionsBySummaryIDRequest struct {
	SummaryID  string `param:"summaryID" example:"bbc15647-0e2e-4f3a-9b2b-a4a918d3f34b"`
	RefNumber  string `query:"refNumber" example:"423423423523523"`
	Limit      int    `query:"limit" example:"10"`
	NextCursor string `query:"nextCursor" example:"2"`
	PrevCursor string `query:"prevCursor" example:"1"`
}

// DetailedTransactionFilterOptions represents filter options for detailed transactions query
type DetailedTransactionFilterOptions struct {
	SummaryID string
	RefNumber string
	Limit     int
	Cursor    *DetailedTransactionCursor
}

// DetailedTransactionCursor represents cursor for pagination
type DetailedTransactionCursor struct {
	ID         string
	IsBackward bool
}

func (c DetailedTransactionCursor) String() string {
	return c.ID
}

// DetailedTransactionOut represents output from repository
type DetailedTransactionOut struct {
	ID                 string
	TransactionID      string
	TransactionDate    time.Time
	RefNumber          string
	TypeTransaction    string
	SourceAccount      string
	DestinationAccount string
	Amount             decimal.Decimal
	Description        string
	Metadata           string
}

// GetCursor returns cursor for pagination
func (d DetailedTransactionOut) GetCursor() string {
	return base64.StdEncoding.EncodeToString([]byte(d.ID))
}

// DetailedTransactionResponse represents API response for detailed transaction
type DetailedTransactionResponse struct {
	TransactionID      string          `json:"transactionID" example:"b14431aa-b3bf-44d0-b287-f504dfb957fe"`
	TransactionDate    string          `json:"transactionDate" example:"2025-10-17"`
	RefNumber          string          `json:"refNumber" example:"423423423523523"`
	TypeTransaction    string          `json:"typeTransaction" example:"SIVEP"`
	SourceAccount      string          `json:"sourceAccount" example:"1310014234242342342"`
	DestinationAccount string          `json:"destinationAccount" example:"42423523523523"`
	Amount             decimal.Decimal `json:"amount" example:"10000"`
	Description        string          `json:"description" example:"NORMAL"`
	Metadata           map[string]any  `json:"metadata" swaggertype:"object"`
}

// ToModelResponse converts DetailedTransactionOut to DetailedTransactionResponse
func (d DetailedTransactionOut) ToModelResponse() DetailedTransactionResponse {
	var metadata map[string]interface{}
	if d.Metadata != "" {
		_ = json.Unmarshal([]byte(d.Metadata), &metadata)
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return DetailedTransactionResponse{
		TransactionID:      d.TransactionID,
		TransactionDate:    d.TransactionDate.Format(constants.DateFormatYYYYMMDD),
		RefNumber:          d.RefNumber,
		TypeTransaction:    d.TypeTransaction,
		SourceAccount:      d.SourceAccount,
		DestinationAccount: d.DestinationAccount,
		Amount:             d.Amount,
		Description:        d.Description,
		Metadata:           metadata,
	}
}

// UpdateMoneyFlowSummaryRequest represents the request body for updating summary
type UpdateMoneyFlowSummaryRequest struct {
	PaymentType       *string `json:"paymentType,omitempty" example:"MF_EARN_DIVEST"`
	TotalTransfer     *string `json:"totalTransfer,omitempty" example:"24000"`
	PapaTransactionID *string `json:"papaTransactionId,omitempty" example:"PAPA-123456"`
	MoneyFlowStatus   *string `json:"moneyFlowStatus,omitempty" example:"COMPLETED"`
	RequestedDate     *string `json:"requestedDate,omitempty" example:"2025-10-21T10:00:00Z"`
	ActualDate        *string `json:"actualDate,omitempty" example:"2025-10-21T15:00:00Z"`
}

// DoUpdateSummaryRequest represents the path parameter for update
type DoUpdateSummaryRequest struct {
	SummaryID string `params:"summaryID" example:"bbc15647-0e2e-4f3a-9b2b-a4a918d3f34b"`
}

// UpdateMoneyFlowSummaryResponse represents the response after updating
type UpdateMoneyFlowSummaryResponse struct {
	Kind      string `json:"kind" example:"moneyFlowCalc"`
	SummaryID string `json:"summaryId" example:"bbc15647-0e2e-4f3a-9b2b-a4a918d3f34b"`
	Message   string `json:"message" example:"Money flow summary updated successfully"`
}

// ToUpdateModel converts request to MoneyFlowSummaryUpdate
func (req UpdateMoneyFlowSummaryRequest) ToUpdateModel() (*MoneyFlowSummaryUpdate, error) {
	update := &MoneyFlowSummaryUpdate{}

	// Payment Type
	if req.PaymentType != nil {
		update.PaymentType = req.PaymentType
	}

	// Total Transfer
	if req.TotalTransfer != nil {
		amount, err := decimal.NewFromString(*req.TotalTransfer)
		if err != nil {
			return nil, fmt.Errorf("invalid totalTransfer format: %w", err)
		}
		update.TotalTransfer = &amount
	}

	// PAPA Transaction ID
	if req.PapaTransactionID != nil {
		update.PapaTransactionID = req.PapaTransactionID
	}

	// Money Flow Status
	if req.MoneyFlowStatus != nil {
		// Validate status
		validStatuses := map[string]bool{
			constants.MoneyFlowStatusPending:    true,
			constants.MoneyFlowStatusSuccess:    true,
			constants.MoneyFlowStatusFailed:     true,
			constants.MoneyFlowStatusInProgress: true,
			constants.MoneyFlowStatusRejected:   true,
		}
		if !validStatuses[*req.MoneyFlowStatus] {
			return nil, fmt.Errorf("invalid moneyFlowStatus: must be PENDING, SUCCESS, IN_PROGRESS, REJECTED or FAILED")
		}
		update.MoneyFlowStatus = req.MoneyFlowStatus
	}

	// Requested Date
	if req.RequestedDate != nil {
		parsedDate, err := time.Parse(time.RFC3339, *req.RequestedDate)
		if err != nil {
			return nil, fmt.Errorf("invalid requestedDate format: must be RFC3339 (ISO8601): %w", err)
		}
		update.RequestedDate = &parsedDate
	}

	// Actual Date
	if req.ActualDate != nil {
		parsedDate, err := time.Parse(time.RFC3339, *req.ActualDate)
		if err != nil {
			return nil, fmt.Errorf("invalid actualDate format: must be RFC3339 (ISO8601): %w", err)
		}
		update.ActualDate = &parsedDate
	}

	return update, nil
}

// Validate checks if at least one field is provided for update
func (req UpdateMoneyFlowSummaryRequest) Validate() error {
	if req.PaymentType == nil &&
		req.TotalTransfer == nil &&
		req.PapaTransactionID == nil &&
		req.MoneyFlowStatus == nil &&
		req.RequestedDate == nil &&
		req.ActualDate == nil {
		return fmt.Errorf("at least one field must be provided for update")
	}
	return nil
}
