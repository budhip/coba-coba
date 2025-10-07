package transformer

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// tupepTransformer is a transformer to acuan transaction for TUPEP transaction type
type tupepTransformer struct {
	baseWalletTransactionTransformer
}

func (t tupepTransformer) Transform(_ context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	if t.config.TransactionValidationConfig.ValidateTUPEPCustomerNum {
		customerNum := getCustomerNumberMetadata(parentWalletTransaction.Metadata)
		if customerNum == "" {
			return nil, common.ErrMissingCustomerNumberFromMetadata
		}
	}

	if t.config.TransactionValidationConfig.ValidateTUPEPLoanType {
		loanType, ok := parentWalletTransaction.Metadata["loanType"].(string)
		if !ok || loanType == "" {
			return nil, common.ErrMissingLoanTypeFromMetadata
		}
	}

	if parentWalletTransaction.DestinationAccountNumber == "" {
		return nil, common.ErrMissingDestinationAccountNumber
	}

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     parentWalletTransaction.AccountNumber,
			ToAccount:       parentWalletTransaction.DestinationAccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "TUPEP",
			OrderType:       "TUP",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Metadata:        parentWalletTransaction.Metadata,
			Description:     parentWalletTransaction.Description,
		},
	}, nil
}
