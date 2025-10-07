package transformer

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type paymdTransformer struct {
	baseWalletTransactionTransformer
}

func (t paymdTransformer) Transform(_ context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	if t.config.TransactionValidationConfig.ValidatePAYMDLoanAccountNumber {
		lan := getLoanAccountNumber(parentWalletTransaction.Metadata)
		if lan == "" {
			return nil, common.ErrMissingLoanAccountNumberFromMetadata
		}
	}

	if t.config.TransactionValidationConfig.ValidatePAYMDLoanIDS {
		loanIds, err := getLoanIds(parentWalletTransaction.Metadata)
		if err != nil {
			return nil, err
		}

		if len(loanIds) == 0 {
			return nil, common.ErrMissingLoanIdsFromMetadata
		}
	}

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     parentWalletTransaction.AccountNumber,
			ToAccount:       t.config.AccountConfig.SystemAccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "PAYMD",
			OrderType:       "PAY",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     parentWalletTransaction.Description,
			Metadata:        parentWalletTransaction.Metadata,
		},
	}, nil
}
