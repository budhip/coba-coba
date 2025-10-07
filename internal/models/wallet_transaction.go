package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
)

type TransactionFlow string
type WalletTransactionStatus string

const (
	TransactionFlowCashIn   TransactionFlow = "cashin"
	TransactionFlowCashOut  TransactionFlow = "cashout"
	TransactionFlowTransfer TransactionFlow = "transfer"
	TransactionFlowRefund   TransactionFlow = "refund"

	WalletTransactionStatusPending WalletTransactionStatus = "PENDING"
	WalletTransactionStatusSuccess WalletTransactionStatus = "SUCCESS"
	WalletTransactionStatusCancel  WalletTransactionStatus = "CANCEL"

	DefaultReversalTimeRangeDay int = 30

	IdempotencyKeyHeader = "X-Idempotency-Key"
	ClientIdHeader       = "X-Client-Id"
)

var (
	AllowedTransactionTypesForLceRollout       = []string{"FPEPD", "FPEPT"}
	AllowedTransactionAmountTypesForLceRollout = []string{"ITDED", "ITDEP"}
)

type WalletMetadata map[string]any

func (e *WalletMetadata) Scan(src interface{}) error {
	var raw []byte
	switch src := src.(type) {
	case string:
		raw = []byte(src)
	case []byte:
		raw = src
	default:
		return fmt.Errorf("type %T not supported by Scan", src)
	}

	return json.Unmarshal(raw, e)
}

func (e WalletMetadata) Value() (value driver.Value, err error) {
	return json.Marshal(e)
}

type CreateWalletTransactionRequest struct {
	IsReserved               bool            `json:"isReserved"`
	AccountNumber            string          `json:"accountNumber" validate:"required"`
	RefNumber                string          `json:"refNumber" validate:"required"`
	TransactionType          string          `json:"transactionType" validate:"required"`
	TransactionFlow          TransactionFlow `json:"transactionFlow" validate:"required,oneof=cashin cashout transfer refund"`
	TransactionTime          string          `json:"transactionTime" validate:"required,iso8601datetime"`
	NetAmount                Amount          `json:"netAmount" validate:"required"`
	Amounts                  []AmountDetail  `json:"amounts" validate:"dive"`
	DestinationAccountNumber string          `json:"destinationAccountNumber"`
	Description              string          `json:"description"`
	Metadata                 WalletMetadata  `json:"metadata"`

	// internal use
	ClientId       string
	IdempotencyKey string
}

func (e CreateWalletTransactionRequest) ToResponse(walletTrx WalletTransaction) WalletTransactionResponse {
	return WalletTransactionResponse{
		Kind:                     "walletTransaction",
		ID:                       walletTrx.ID,
		Status:                   walletTrx.Status,
		AccountNumber:            e.AccountNumber,
		RefNumber:                e.RefNumber,
		TransactionType:          e.TransactionType,
		TransactionFlow:          e.TransactionFlow,
		TransactionTime:          e.TransactionTime,
		NetAmount:                e.NetAmount,
		Amounts:                  e.Amounts,
		DestinationAccountNumber: e.DestinationAccountNumber,
		Description:              e.Description,
		Metadata:                 e.Metadata,
	}
}

func (e CreateWalletTransactionRequest) ToNewWalletTransaction() NewWalletTransaction {
	status := WalletTransactionStatusSuccess
	if e.IsReserved {
		status = WalletTransactionStatusPending
	}

	trxTime, _ := time.Parse(time.RFC3339, e.TransactionTime)

	return NewWalletTransaction{
		ID:                       uuid.New().String(),
		Status:                   status,
		Amounts:                  e.Amounts,
		AccountNumber:            e.AccountNumber,
		RefNumber:                e.RefNumber,
		TransactionType:          e.TransactionType,
		TransactionFlow:          e.TransactionFlow,
		TransactionTime:          trxTime,
		NetAmount:                e.NetAmount,
		DestinationAccountNumber: e.DestinationAccountNumber,
		Description:              e.Description,
		Metadata:                 e.Metadata,
	}
}

type WalletTransactionResponse struct {
	Kind                     string                  `json:"kind"`
	ID                       string                  `json:"id"`
	Status                   WalletTransactionStatus `json:"status"`
	AccountNumber            string                  `json:"accountNumber"`
	RefNumber                string                  `json:"refNumber"`
	TransactionType          string                  `json:"transactionType"`
	TransactionFlow          TransactionFlow         `json:"transactionFlow"`
	TransactionTime          string                  `json:"transactionTime"`
	NetAmount                Amount                  `json:"netAmount"`
	Amounts                  []AmountDetail          `json:"amounts"`
	DestinationAccountNumber string                  `json:"destinationAccountNumber"`
	Description              string                  `json:"description"`
	Metadata                 WalletMetadata          `json:"metadata"`
}

// WalletTransaction representing wallet_transaction table
type WalletTransaction struct {
	ID                       string
	Status                   WalletTransactionStatus
	AccountNumber            string
	DestinationAccountNumber string
	RefNumber                string
	TransactionType          string
	TransactionTime          time.Time
	TransactionFlow          TransactionFlow
	NetAmount                Amount
	Amounts                  Amounts
	Description              string
	Metadata                 WalletMetadata
	CreatedAt                time.Time
}

func (e WalletTransaction) ToResponse() WalletTransactionResponse {
	return WalletTransactionResponse{
		Kind:                     "walletTransaction",
		ID:                       e.ID,
		Status:                   e.Status,
		AccountNumber:            e.AccountNumber,
		DestinationAccountNumber: e.DestinationAccountNumber,
		RefNumber:                e.RefNumber,
		TransactionType:          e.TransactionType,
		TransactionFlow:          e.TransactionFlow,
		TransactionTime:          e.TransactionTime.In(common.GetLocation()).Format(common.DateFormatYYYYMMDDWithTime),
		NetAmount:                e.NetAmount,
		Amounts:                  e.Amounts,
		Description:              e.Description,
		Metadata:                 e.Metadata,
	}
}

func (e WalletTransaction) GetCursor() string {
	wtc := WalletTrxCursor{
		TransactionTime: e.TransactionTime,
		Id:              e.ID,
	}
	return wtc.String()
}

func (e WalletTransaction) ToModelResponse() ListWalletTrxByAccountNumberResponse {
	amount := e.NetAmount
	if amount.Currency == "" {
		amount.Currency = "IDR"
	}

	return ListWalletTrxByAccountNumberResponse{
		Kind:                "walletTransaction",
		TransactionDate:     common.FormatDatetimeToStringInLocalTime(e.TransactionTime, common.DateFormatYYYYMMDD),
		TransactionTime:     e.TransactionTime.In(common.GetLocation()).Format(common.DateFormatYYYYMMDDWithTimeAndOffset),
		TransactionType:     e.TransactionType,
		Amount:              amount,
		Amounts:             e.Amounts,
		Status:              e.Status,
		TransactionWalletId: e.ID,
		RefNumber:           e.RefNumber,
		Description:         e.Description,
		TransactionFlow:     e.TransactionFlow,
		Metadata:            e.Metadata,
	}
}

type ErrWalletTransaction struct {
	LineNumb        string
	AccountNumber   string
	RefNumber       string
	TransactionType string
	Error           string
}

// NewWalletTransaction is payload to create new wallet transaction
type NewWalletTransaction struct {
	ID                       string
	AccountNumber            string
	RefNumber                string
	TransactionType          string
	TransactionFlow          TransactionFlow
	TransactionTime          time.Time
	NetAmount                Amount
	Amounts                  Amounts
	Status                   WalletTransactionStatus
	DestinationAccountNumber string
	Description              string
	Metadata                 WalletMetadata
}

func (nw NewWalletTransaction) ToWalletTransaction() WalletTransaction {
	if nw.Metadata == nil {
		nw.Metadata = WalletMetadata{}
	}

	return WalletTransaction{
		ID:                       nw.ID,
		Status:                   nw.Status,
		AccountNumber:            nw.AccountNumber,
		DestinationAccountNumber: nw.DestinationAccountNumber,
		RefNumber:                nw.RefNumber,
		TransactionType:          nw.TransactionType,
		TransactionFlow:          nw.TransactionFlow,
		TransactionTime:          nw.TransactionTime,
		NetAmount:                nw.NetAmount,
		Amounts:                  nw.Amounts,
		Description:              nw.Description,
		Metadata:                 nw.Metadata,
		CreatedAt:                time.Now(),
	}
}

// UpdateStatusWalletTransactionRequest is DTO object from handler
type UpdateStatusWalletTransactionRequest struct {
	TransactionId      string `json:"-" validate:"required"`
	Action             string `json:"action" example:"commit" validate:"required,oneof=commit cancel"`
	RawTransactionTime string `json:"transactionTime" validate:"iso8601datetime"`

	TransactionTime time.Time `json:"-"`

	// internal use
	ClientId string
}

func (e *UpdateStatusWalletTransactionRequest) TransformTransactionTime() error {
	if e.RawTransactionTime != "" {
		transactionTime, errParse := time.Parse(time.RFC3339, e.RawTransactionTime)
		if errParse != nil {
			return GetErrMap(ErrKeyTransactionTimeIso8601Datetime, fmt.Sprintf("invalid transactionTime: %s", errParse))
		}

		e.TransactionTime = transactionTime
	}

	return nil
}

func (e UpdateStatusWalletTransactionRequest) ToResponse(walletTrx WalletTransaction) UpdateStatusWalletTransactionResponse {
	return UpdateStatusWalletTransactionResponse{
		Kind:          "walletTransaction",
		TransactionId: walletTrx.ID,
		Status:        string(walletTrx.Status),
	}
}

type UpdateStatusWalletTransactionResponse struct {
	Kind          string `json:"kind" example:"walletTransaction"`
	TransactionId string `json:"transactionId" example:"41d03147-c017-4176-8a1a-0b7ec735cc29"`
	Status        string `json:"status" example:"SUCCESS"`
}

// WalletTransactionUpdate only used when repository call update
type WalletTransactionUpdate struct {
	Status          *WalletTransactionStatus
	TransactionTime *time.Time
}
