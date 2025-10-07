package transformer

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type mfwrqTransformer struct {
	baseWalletTransactionTransformer
}

func (t mfwrqTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	account, errGetAccount := t.accountRepository.GetCachedAccount(
		ctx,
		parentWalletTransaction.AccountNumber,
	)
	if errGetAccount != nil {
		return nil, errGetAccount
	}

	entityCode := account.Entity
	if entityCode == "" {
		return nil, common.ErrMissingEntityFromAccount
	}

	metadata := parentWalletTransaction.Metadata
	metadata["entity"] = t.config.AccountConfig.MapAccountEntity[entityCode]

	return []models.TransactionReq {
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     parentWalletTransaction.AccountNumber,
			ToAccount:       t.config.AccountConfig.SystemAccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "MFWRQ",
			OrderType:       "MFW",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     parentWalletTransaction.Description,
			Metadata:        metadata,
		},
	}, nil
}