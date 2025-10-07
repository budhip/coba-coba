package transformer

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type rpyadTransformer struct {
	baseWalletTransactionTransformer
}

func (t rpyadTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	investedAccountNumber, err := t.accountingClient.GetInvestedAccountNumber(ctx, parentWalletTransaction.AccountNumber)
	if err != nil {
		return nil, err
	}

	account, err := t.accountRepository.GetCachedAccount(ctx, parentWalletTransaction.AccountNumber)
	if err != nil {
		return res, err
	}

	entityCode := account.Entity
	if entityCode == "" {
		return res, common.ErrMissingEntityFromAccount
	}

	metadata := parentWalletTransaction.Metadata
	metadata = t.MutateMetadataByAccountEntity(entityCode, metadata)

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     investedAccountNumber,
			ToAccount:       parentWalletTransaction.AccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "RPYAD",
			OrderType:       "RPY",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     parentWalletTransaction.Description,
			Metadata:        parentWalletTransaction.Metadata,
		},
	}, nil
}
