package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/go-multierror"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"

	"bitbucket.org/Amartha/go-acuan-lib/model"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	TransactionIDPrefix             = "TRX"
	TransactionIDManualPrefix       = "TRX-MANUAL"
	WalletTransactionIDManualPrefix = "MANUAL"
	TransactionRequestCommitStatus  = "commit"
	TransactionRequestCancelStatus  = "cancel"
	MaxRowTransactionFile           = 500000
	TransactionStatusSuccessNum     = "1"
	TransactionStatusCancelNum      = "2"
	TransactionStatusPendingNum     = "0"
)

type TransactionStoreProcessType int

const (
	TransactionStoreProcessNormal TransactionStoreProcessType = iota
	TransactionStoreProcessReserved
)

type Transaction struct {
	ID                         uint64              `json:"id"`
	TransactionID              string              `json:"transactionId"`
	TransactionDate            time.Time           `json:"transactionDate"`
	FromAccount                string              `json:"fromAccount"`
	SourceEntity               string              `json:"sourceEntity,omitempty"`
	FromAccountProductTypeName string              `json:"fromAccountProductTypeName"`
	FromAccountName            string              `json:"fromAccountName"`
	ToAccount                  string              `json:"toAccount"`
	DestinationEntity          string              `json:"destinationEntity,omitempty"`
	ToAccountProductTypeName   string              `json:"toAccountProductTypeName"`
	ToAccountName              string              `json:"toAccountName"`
	FromNarrative              string              `json:"fromNarrative"`
	ToNarrative                string              `json:"toNarrative"`
	Amount                     decimal.NullDecimal `json:"amount"`
	Status                     string              `json:"status"`
	Method                     string              `json:"method"`
	TypeTransaction            string              `json:"typeTransaction"`
	Description                string              `json:"description"`
	RefNumber                  string              `json:"refNumber"`
	Metadata                   string              `json:"metadata"`
	CreatedAt                  time.Time           `json:"createdAt"`
	UpdatedAt                  time.Time           `json:"updatedAt"`
	DeletedAt                  *time.Time          `json:"deletedAt,omitempty"`
	TotalData                  int                 `json:"totalData,omitempty"`
	OrderTime                  time.Time           `json:"orderTime"`
	OrderType                  string              `json:"orderType"`
	TransactionTime            time.Time           `json:"transactionTime"`
	Currency                   string              `json:"currency"`
}

func (e *Transaction) ToAcuanLibTransaction() (*model.Transaction, error) {
	var metadata any
	if e.Metadata != "" {
		err := json.Unmarshal([]byte(e.Metadata), &metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %v", err)
		}
	}

	transactionId, err := uuid.Parse(e.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction id: %v", err)
	}

	rawStatus, err := strconv.ParseUint(e.Status, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse status: %v", err)
	}

	transactionType := model.TransactionType(e.TypeTransaction)
	method := model.TransactionMethod(e.Method)
	status := model.TransactionStatus(uint8(rawStatus))

	return &model.Transaction{
		Id:                   &transactionId,
		Amount:               e.Amount.Decimal,
		Currency:             e.Currency,
		SourceAccountId:      e.FromAccount,
		SourceEntity:         e.SourceEntity,
		DestinationAccountId: e.ToAccount,
		DestinationEntity:    e.DestinationEntity,
		Description:          e.Description,
		Method:               method,
		TransactionType:      transactionType,
		TransactionTime: model.AcuanTime{
			Time: &e.TransactionTime,
		},
		Status: status,
		Meta:   metadata,
	}, nil
}

func (e *Transaction) ToAcuanNotificationMessage(statusNotification StatusTransactionNotification, message, clientID string) (*TransactionNotificationPayload, error) {
	acuanLibTransaction, err := e.ToAcuanLibTransaction()
	if err != nil {
		return nil, err
	}

	orderType := model.OrderType(e.OrderType)

	return &TransactionNotificationPayload{
		Identifier: e.RefNumber,
		Status:     statusNotification,
		AcuanData: model.Payload[model.DataOrder]{
			Headers: model.Headers{
				SourceSystem: "go-fp-transaction",
			},
			Body: model.Body[model.DataOrder]{
				Data: model.DataOrder{
					Order: model.Order{
						OrderTime: model.AcuanTime{
							Time: &e.OrderTime,
						},
						OrderType: orderType,
						RefNumber: e.RefNumber,
						Transactions: []model.Transaction{
							*acuanLibTransaction,
						},
					},
				},
			},
		},
		Message:  message,
		ClientID: clientID,
	}, nil
}

func (e *Transaction) ToReconFormat() []string {
	return []string{
		fmt.Sprint(e.ID),
		common.FormatDatetimeToString(e.TransactionDate, common.DateFormatYYYYMMDD),
		e.FromAccount,
		e.FromNarrative,
		e.ToAccount,
		e.ToNarrative,
		fmt.Sprint(e.Amount.Decimal),
		e.Status,
		e.Method,
		e.TypeTransaction,
		e.Description,
		e.RefNumber,
		common.FormatDatetimeToString(e.CreatedAt, common.DateFormatYYYYMMDD),
		common.FormatDatetimeToString(e.UpdatedAt, common.DateFormatYYYYMMDD),
		e.Metadata,
		e.TransactionID,
	}
}

func (e *Transaction) ToGetTransactionOut(mapOrderType, mapTransactionType map[string]string) GetTransactionOut {
	return GetTransactionOut{
		DatabaseID:                 e.ID,
		TransactionID:              e.TransactionID,
		FromAccount:                e.FromAccount,
		FromAccountName:            e.FromAccountName,
		FromAccountProductTypeName: e.FromAccountProductTypeName,
		ToAccount:                  e.ToAccount,
		ToAccountName:              e.ToAccountName,
		ToAccountProductTypeName:   e.ToAccountProductTypeName,
		Currency:                   e.Currency,
		Amount:                     e.Amount.Decimal,
		RefNumber:                  e.RefNumber,
		OrderType:                  e.OrderType,
		OrderTypeName:              mapOrderType[e.OrderType],
		TransactionType:            e.TypeTransaction,
		TransactionTypeName:        mapTransactionType[e.TypeTransaction],
		Method:                     e.Method,
		TransactionTime:            e.TransactionTime,
		TransactionDate:            e.TransactionDate,
		Status:                     e.Status,
		Description:                e.Description,
		Metadata:                   e.Metadata,
		CreatedAt:                  e.CreatedAt,
		UpdatedAt:                  e.UpdatedAt,
	}
}

func (e *Transaction) GetStatusString() string {
	title, ok := MapTransactionStatus[TransactionStatus(e.Status)]
	if !ok {
		return fmt.Sprint("UNKNOWN: ", e.Status)
	}

	return title
}

func (e *Transaction) IsSuccess() bool {
	return e.Status == string(TransactionStatusSuccess)
}

func (e *Transaction) IsCancel() bool {
	return e.Status == string(TransactionStatusCancel)
}

func (e *Transaction) IsPending() bool {
	return e.Status == string(TransactionStatusPending)
}

func (e *Transaction) ToUpdateStatusReservedTransactionResponse() UpdateStatusReservedTransactionResponse {
	return UpdateStatusReservedTransactionResponse{
		Kind:          kindTransaction,
		TransactionId: e.TransactionID,
		Status:        e.GetStatusString(),
	}
}

type TransactionStreamResult struct {
	Data Transaction
	Err  error
}

type DoCreateTransactionRequest struct {
	IsReserved      bool            `json:"isReserved"`
	FromAccount     string          `json:"fromAccount" validate:"required"`
	ToAccount       string          `json:"toAccount" validate:"required"`
	Amount          decimal.Decimal `json:"amount" validate:"required"`
	Method          string          `json:"method"`
	TransactionType string          `json:"transactionType" validate:"required"`
	TransactionTime time.Time       `json:"transactionTime" validate:"required"`
	OrderType       string          `json:"orderType" validate:"required"`
	Description     string          `json:"description"`
	RefNumber       string          `json:"refNumber" validate:"required"`
	Metadata        map[string]any  `json:"metadata"`
}

// ToTransactionReq convert DoCreateTransactionRequest to TransactionReq.
// TransactionReq is used as parameter to store transaction from service
func (r *DoCreateTransactionRequest) ToTransactionReq() TransactionReq {
	status := TransactionStatusSuccess
	if r.IsReserved {
		status = TransactionStatusPending
	}

	return TransactionReq{
		TransactionID:   uuid.New().String(),
		FromAccount:     r.FromAccount,
		ToAccount:       r.ToAccount,
		Status:          string(status),
		TransactionDate: common.FormatDatetimeToStringInLocalTime(r.TransactionTime, common.DateFormatYYYYMMDD),
		Amount:          decimal.NewNullDecimal(r.Amount),
		Method:          r.Method,
		TypeTransaction: r.TransactionType,
		Description:     r.Description,
		RefNumber:       r.RefNumber,
		Metadata:        r.Metadata,
		OrderTime:       time.Now(),
		OrderType:       r.OrderType,
		TransactionTime: r.TransactionTime,
		Currency:        "IDR",
	}
}

type TransactionReq struct {
	TransactionID   string              `json:"transactionId"`
	FromAccount     string              `json:"fromAccount"`
	ToAccount       string              `json:"toAccount"`
	FromNarrative   string              `json:"fromNarrative"`
	ToNarrative     string              `json:"toNarrative"`
	TransactionDate string              `json:"transactionDate"`
	Amount          decimal.NullDecimal `json:"amount"`
	Status          string              `json:"status"`
	Method          string              `json:"method"`
	TypeTransaction string              `json:"typeTransaction"`
	Description     string              `json:"description"`
	RefNumber       string              `json:"refNumber"`
	Metadata        interface{}         `json:"metadata"`
	OrderTime       time.Time           `json:"orderTime"`
	OrderType       string              `json:"orderType"`
	TransactionTime time.Time           `json:"transactionTime"`
	Currency        string              `json:"currency"`
}

func (req *TransactionReq) ToRequest() (en Transaction, err error) {
	var trxDate time.Time
	trxDate, err = common.ParseStringDateToDateWithTimeNow(common.DateFormatYYYYMMDD, req.TransactionDate)
	if err != nil {
		return en, common.WrapError{
			Causer: fmt.Errorf("format must be YYYY-MM-DD: %v", en.FromNarrative),
			Err:    common.ErrInvalidFormatDate,
		}
	}

	byteMetadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return en, err
	}

	if req.TransactionID == "" {
		req.TransactionID = uuid.New().String()
	}

	en = Transaction{
		TransactionID:   req.TransactionID,
		TransactionDate: trxDate,
		FromAccount:     req.FromAccount,
		ToAccount:       req.ToAccount,
		FromNarrative:   req.FromNarrative,
		ToNarrative:     req.ToNarrative,
		Amount:          req.Amount,
		Status:          req.Status,
		Method:          req.Method,
		TypeTransaction: req.TypeTransaction,
		Description:     req.Description,
		RefNumber:       req.RefNumber,
		Metadata:        string(byteMetadata),
		OrderTime:       req.OrderTime,
		OrderType:       req.OrderType,
		TransactionTime: req.TransactionTime,
		Currency:        req.Currency,
	}

	return
}

func (req *TransactionReq) ToAcuanLibTransaction() (*model.Transaction, error) {
	transactionId, err := uuid.Parse(req.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction id: %v", err)
	}

	rawStatus, err := strconv.ParseUint(req.Status, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse status: %v", err)
	}

	transactionType := model.TransactionType(req.TypeTransaction)
	method := model.TransactionMethod(req.Method)
	status := model.TransactionStatus(uint8(rawStatus))

	return &model.Transaction{
		Id:                   &transactionId,
		Amount:               req.Amount.Decimal,
		Currency:             req.Currency,
		SourceAccountId:      req.FromAccount,
		DestinationAccountId: req.ToAccount,
		Description:          req.Description,
		Method:               method,
		TransactionType:      transactionType,
		TransactionTime: model.AcuanTime{
			Time: &req.TransactionTime,
		},
		Status: status,
		Meta:   req.Metadata,
	}, nil
}

type UpdateStatusReservedTransactionRequest struct {
	TransactionId string `json:"-" validate:"required"`
	Status        string `json:"status" example:"commit" validate:"required,oneof=commit cancel"`
}

type UpdateStatusReservedTransactionResponse struct {
	Kind          string `json:"kind" example:"transaction"`
	TransactionId string `json:"transactionId" example:"41d03147-c017-4176-8a1a-0b7ec735cc29"`
	Status        string `json:"status" example:"SUCCESS"`
}

type CreateOrderTransactionRequest struct {
	ID                   *uuid.UUID      `json:"id" validate:"required"`
	Amount               decimal.Decimal `json:"amount" validate:"required"`
	Currency             string          `json:"currency" validate:"required"`
	SourceAccountId      string          `json:"sourceAccountId" validate:"required"`
	DestinationAccountId string          `json:"destinationAccountId" validate:"required"`
	Description          string          `json:"description"`
	Method               string          `json:"method"`
	TransactionType      string          `json:"transactionType" validate:"required"`
	TransactionTime      *time.Time      `json:"transactionTime" validate:"required"`
	Status               *int            `json:"status" validate:"required"`
	Meta                 any             `json:"meta"`
}

type CreateOrderRequest struct {
	OrderTime    *time.Time                      `json:"orderTime" validate:"required"`
	OrderType    string                          `json:"orderType" validate:"required"`
	RefNumber    string                          `json:"refNumber" validate:"required"`
	Transactions []CreateOrderTransactionRequest `json:"transactions" validate:"required,gt=0,dive"`
}

func (e CreateOrderRequest) ToTransactionReqs() (result []TransactionReq) {
	for _, v := range e.Transactions {
		var transactionID string
		if v.ID != nil {
			transactionID = v.ID.String()
		}
		orderTime := common.Now()
		if e.OrderTime != nil {
			orderTime = *e.OrderTime
		}
		trxTime := common.Now()
		if v.TransactionTime != nil {
			trxTime = *v.TransactionTime
		}
		status := 0
		if v.Status != nil {
			status = *v.Status
		}

		orderReq := TransactionReq{
			TransactionID:   transactionID,
			FromAccount:     v.SourceAccountId,
			ToAccount:       v.DestinationAccountId,
			FromNarrative:   "",
			ToNarrative:     "",
			TransactionDate: common.FormatDatetimeToStringInLocalTime(trxTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(v.Amount),
			Status:          fmt.Sprint(status),
			Method:          v.Method,
			TypeTransaction: v.TransactionType,
			Description:     v.Description,
			RefNumber:       e.RefNumber,
			Metadata:        v.Meta,
			OrderTime:       orderTime,
			OrderType:       e.OrderType,
			TransactionTime: trxTime,
			Currency:        v.Currency,
		}
		result = append(result, orderReq)
	}

	return
}

func (e CreateOrderRequest) ToCreateOrderResponse() CreateOrderResponse {
	return CreateOrderResponse{
		Kind:         "order",
		OrderTime:    e.OrderTime,
		OrderType:    e.OrderType,
		RefNumber:    e.RefNumber,
		Transactions: e.Transactions,
	}
}

type CreateOrderResponse struct {
	Kind         string                          `json:"kind"`
	OrderTime    *time.Time                      `json:"orderTime"`
	OrderType    string                          `json:"orderType"`
	RefNumber    string                          `json:"refNumber"`
	Transactions []CreateOrderTransactionRequest `json:"transactions"`
}

func SerializeTransactionReqToNotification(inputReqs []TransactionReq, statusNotification StatusTransactionNotification, message, clientID string) (*TransactionNotificationPayload, error) {
	var acuanLibTransactions []model.Transaction

	var errs *multierror.Error
	for _, req := range inputReqs {
		acuanTx, err := req.ToAcuanLibTransaction()
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		acuanLibTransactions = append(acuanLibTransactions, *acuanTx)
	}

	err := errs.ErrorOrNil()
	if err != nil {
		return nil, err
	}

	if len(acuanLibTransactions) == 0 {
		return nil, fmt.Errorf("acuan lib transaction lenght is 0")
	}

	firstRawInput := inputReqs[0]
	return &TransactionNotificationPayload{
		Identifier: firstRawInput.RefNumber,
		Status:     statusNotification,
		AcuanData: model.Payload[model.DataOrder]{
			Headers: model.Headers{
				SourceSystem: "go-fp-transaction",
			},
			Body: model.Body[model.DataOrder]{
				Data: model.DataOrder{
					Order: model.Order{
						OrderTime: model.AcuanTime{
							Time: &firstRawInput.OrderTime,
						},
						OrderType:    model.OrderType(firstRawInput.OrderType),
						RefNumber:    firstRawInput.RefNumber,
						Transactions: acuanLibTransactions,
					},
				},
			},
		},
		Message:  message,
		ClientID: clientID,
	}, nil
}

type ReportRepayment struct {
	TransactionDate time.Time       `json:"transactionDate"`
	Outstanding     decimal.Decimal `json:"outstanding"`
	Principal       decimal.Decimal `json:"principal"`
	Amartha         decimal.Decimal `json:"amartha"`
	Lender          decimal.Decimal `json:"lender"`
	PPN             decimal.Decimal `json:"ppn"`
	PPh             decimal.Decimal `json:"pph"`
	Total           decimal.Decimal `json:"total"`
}
type DoGetReportRepaymentResponse struct {
	Kind string                    `json:"kind"`
	Data []ReportRepaymentResponse `json:"data"`
}

type ReportRepaymentResponse struct {
	TransactionDate string          `json:"transactionDate"`
	Outstanding     decimal.Decimal `json:"outstanding"`
	Principal       decimal.Decimal `json:"principal"`
	Amartha         decimal.Decimal `json:"amartha"`
	Lender          decimal.Decimal `json:"lender"`
	PPN             decimal.Decimal `json:"ppn"`
	PPh             decimal.Decimal `json:"pph"`
	Total           decimal.Decimal `json:"total"`
}

type ReportRepayments []ReportRepayment

func (in ReportRepayments) ToResponse() DoGetReportRepaymentResponse {
	resp := make([]ReportRepaymentResponse, len(in))
	for i, ts := range in {
		resp[i] = ReportRepaymentResponse{
			TransactionDate: ts.TransactionDate.Format("2006-01-02"),
			Outstanding:     ts.Outstanding,
			Principal:       ts.Principal,
			Amartha:         ts.Amartha,
			Lender:          ts.Lender,
			PPN:             ts.PPN,
			PPh:             ts.PPh,
			Total:           ts.Total,
		}
	}
	return DoGetReportRepaymentResponse{
		Kind: "transaction-summary",
		Data: resp,
	}
}
