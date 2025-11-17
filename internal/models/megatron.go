package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// TransformRequest adalah request untuk transform wallet transaction
type TransformRequest struct {
	// Parent wallet transaction info
	ParentTransaction WalletTransactionInput `json:"parentTransaction"`

	// Amount to transform (bisa dari NetAmount atau dari Amounts)
	Amount AmountInput `json:"amount"`

	// Transaction type yang akan di-transform
	TransactionType string `json:"transactionType"`

	// Context information
	Context TransformContext `json:"context"`
}

type WalletTransactionInput struct {
	ID                       string                 `json:"id"`
	AccountNumber            string                 `json:"accountNumber"`
	DestinationAccountNumber string                 `json:"destinationAccountNumber"`
	RefNumber                string                 `json:"refNumber"`
	TransactionType          string                 `json:"transactionType"`
	TransactionFlow          string                 `json:"transactionFlow"`
	TransactionTime          time.Time              `json:"transactionTime"`
	Description              string                 `json:"description"`
	Metadata                 map[string]interface{} `json:"metadata"`
	Status                   string                 `json:"status"`

	// Account info (akan di-populate dari database)
	Account AccountInfo `json:"account,omitempty"`
}

type AccountInfo struct {
	AccountNumber   string `json:"accountNumber"`
	Name            string `json:"name"`
	Entity          string `json:"entity"`
	CategoryCode    string `json:"categoryCode"`
	SubCategoryCode string `json:"subCategoryCode"`
}

type AmountInput struct {
	Value    decimal.Decimal `json:"value"`
	Currency string          `json:"currency"`
}

type TransformContext struct {
	// Config yang dibutuhkan (akan di-inject dari go-fp-transaction)
	SystemAccountNumber                               string            `json:"systemAccountNumber"`
	AccountNumberInsurancePremiumDisbursementByEntity map[string]string `json:"accountNumberInsurancePremiumDisbursementByEntity,omitempty"`
	MapAccountEntity                                  map[string]string `json:"mapAccountEntity,omitempty"`

	// Additional context
	ClientID      string `json:"clientId,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
	RequestID     string `json:"requestId,omitempty"`
}

// TransformResponse adalah response dari transformation
type TransformResponse struct {
	// Hasil transformasi
	Transactions []TransactionOutput `json:"transactions"`

	// Metadata
	Metadata TransformMetadata `json:"metadata"`

	// Validation warnings (jika ada)
	Warnings []string `json:"warnings,omitempty"`
}

type TransactionOutput struct {
	TransactionID   string                 `json:"transactionId"`
	FromAccount     string                 `json:"fromAccount"`
	ToAccount       string                 `json:"toAccount"`
	FromNarrative   string                 `json:"fromNarrative"`
	ToNarrative     string                 `json:"toNarrative"`
	TransactionDate string                 `json:"transactionDate"`
	Amount          decimal.Decimal        `json:"amount"`
	Status          string                 `json:"status"`
	Method          string                 `json:"method"`
	TypeTransaction string                 `json:"typeTransaction"`
	Description     string                 `json:"description"`
	RefNumber       string                 `json:"refNumber"`
	Metadata        map[string]interface{} `json:"metadata"`
	OrderTime       time.Time              `json:"orderTime"`
	OrderType       string                 `json:"orderType"`
	TransactionTime time.Time              `json:"transactionTime"`
	Currency        string                 `json:"currency"`
}

type TransformMetadata struct {
	RuleID          string `json:"ruleId"`
	RuleName        string `json:"ruleName"`
	RuleVersion     int    `json:"ruleVersion"`
	ExecutionTimeMs int    `json:"executionTimeMs"`
}

// BatchTransformRequest untuk transform multiple amounts sekaligus
type BatchTransformRequest struct {
	ParentTransaction WalletTransactionInput `json:"parentTransaction"`
	Transforms        []TransformItem        `json:"transforms"`
	Context           TransformContext       `json:"context"`
}

type TransformItem struct {
	Amount          AmountInput `json:"amount"`
	TransactionType string      `json:"transactionType"`
}

type BatchTransformResponse struct {
	Transactions []TransactionOutput `json:"transactions"`
	Errors       []TransformError    `json:"errors,omitempty"`
	Metadata     TransformMetadata   `json:"metadata"`
}

type TransformError struct {
	TransactionType string `json:"transactionType"`
	Error           string `json:"error"`
	Code            string `json:"code"`
}

// Rule Management APIs

type CreateRuleRequest struct {
	TransactionType string                 `json:"transactionType"`
	RuleName        string                 `json:"ruleName"`
	Description     string                 `json:"description"`
	Config          map[string]interface{} `json:"config"`
	Tags            map[string]interface{} `json:"tags"`
	CreatedBy       string                 `json:"createdBy"`
}

type UpdateRuleRequest struct {
	Config      map[string]interface{} `json:"config"`
	ChangeNotes string                 `json:"changeNotes"`
	UpdatedBy   string                 `json:"updatedBy"`
}

type RuleResponse struct {
	ID              string                 `json:"id"`
	TransactionType string                 `json:"transactionType"`
	RuleName        string                 `json:"ruleName"`
	Description     string                 `json:"description"`
	Version         int                    `json:"version"`
	IsActive        bool                   `json:"isActive"`
	Config          map[string]interface{} `json:"config"`
	Tags            map[string]interface{} `json:"tags"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

type ListRulesRequest struct {
	TransactionTypes []string               `json:"transactionTypes,omitempty"`
	IsActive         *bool                  `json:"isActive,omitempty"`
	Tags             map[string]interface{} `json:"tags,omitempty"`
	Limit            int                    `json:"limit"`
	Offset           int                    `json:"offset"`
}

type ListRulesResponse struct {
	Rules []RuleResponse `json:"rules"`
	Total int            `json:"total"`
}

// API Routes:
// POST   /api/v1/transform                    - Transform single transaction
// POST   /api/v1/transform/batch               - Transform multiple transactions
// GET    /api/v1/rules                         - List all rules
// GET    /api/v1/rules/:transactionType        - Get specific rule
// POST   /api/v1/rules                         - Create new rule
// PUT    /api/v1/rules/:transactionType        - Update rule (creates new version)
// DELETE /api/v1/rules/:transactionType        - Deactivate rule
// GET    /api/v1/rules/:transactionType/versions - Get rule version history
