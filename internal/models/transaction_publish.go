package models

import (
	"fmt"

	goAcuanLibModel "bitbucket.org/Amartha/go-acuan-lib/model"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/acuanclient"

	"github.com/shopspring/decimal"
)

var kindTransaction = "transaction"

type (
	DoPublishTransactionRequest struct {
		FromAccount     string      `json:"fromAccount" validate:"required,numeric" example:"666"`
		ToAccount       string      `json:"toAccount" validate:"required,numeric" example:"777"`
		Amount          string      `json:"amount" validate:"required,numeric" example:"500000.00"`
		Method          string      `json:"method" validate:"noStartEndSpaces" example:"BANK_TRANFER"`
		TransactionType string      `json:"transactionType" validate:"required,alpha,noStartEndSpaces" example:"TOPUP"`
		TransactionDate string      `json:"transactionDate" validate:"required,date" example:"2006-12-02"`
		OrderType       string      `json:"orderType" validate:"required,alpha,noStartEndSpaces" example:"TOPUP"`
		RefNumber       string      `json:"refNumber" validate:"noStartEndSpaces" example:"97075538-e0ac-460f-b5c2-61c6e14fc72d"`
		Description     string      `json:"description" validate:"nospecial,noStartEndSpaces" example:"topup lenderId 666 via BANK TRANSFER"`
		Metadata        interface{} `json:"metadata"`
	}

	DoPublishTransactionResponse struct {
		Kind            string      `json:"kind" example:"transaction"`
		FromAccount     string      `json:"fromAccount" example:"666"`
		ToAccount       string      `json:"toAccount" example:"777"`
		Amount          string      `json:"amount" example:"500000.00"`
		Method          string      `json:"method" example:"BANK_TRANFER"`
		TransactionType string      `json:"transactionType" example:"TOPUP"`
		TransactionDate string      `json:"transactionDate" example:"2006-12-02"`
		OrderType       string      `json:"orderType" example:"TOPUP"`
		RefNumber       string      `json:"refNumber" example:"97075538-e0ac-460f-b5c2-61c6e14fc72d"`
		Description     string      `json:"description" example:"topup lenderId 666 via BANK TRANSFER"`
		Metadata        interface{} `json:"metadata,omitempty"`
	}
)

func (c *DoPublishTransactionRequest) ValidateToRequest() (req acuanclient.PublishTransactionRequest, err error) {
	transactionDate, err := common.ParseStringDateToDateWithTimeNow(common.DateFormatYYYYMMDD, c.TransactionDate)
	if err != nil {
		return req, GetErrMap(ErrKeyInvalidFormatDate, fmt.Sprintf("date %s format must be YYYY-MM-DD", c.TransactionDate))
	}

	amount, err := decimal.NewFromString(c.Amount)
	if err != nil {
		return req, GetErrMap(ErrKeyInvalidFormatAmount, err.Error())
	}

	req = acuanclient.PublishTransactionRequest{
		FromAccount:     c.FromAccount,
		ToAccount:       c.ToAccount,
		Amount:          amount,
		Method:          goAcuanLibModel.TransactionMethod(c.Method),
		TransactionType: goAcuanLibModel.TransactionType(c.TransactionType),
		TransactionTime: transactionDate,
		OrderType:       c.OrderType,
		RefNumber:       c.RefNumber,
		Description:     c.Description,
		Metadata:        c.Metadata,
	}

	return
}

func (res *DoPublishTransactionRequest) ToPublishResponse() DoPublishTransactionResponse {
	return DoPublishTransactionResponse{
		Kind:            kindTransaction,
		FromAccount:     res.FromAccount,
		ToAccount:       res.ToAccount,
		Amount:          res.Amount,
		Method:          res.Method,
		TransactionType: res.TransactionType,
		TransactionDate: res.TransactionDate,
		OrderType:       res.OrderType,
		RefNumber:       res.RefNumber,
		Description:     res.Description,
		Metadata:        res.Metadata,
	}
}
