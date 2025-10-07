package transformer

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type dsbabTransformer struct {
	baseWalletTransactionTransformer
}

func (t dsbabTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	metadata := parentWalletTransaction.Metadata
	metadata["loanAccountNumber"] = parentWalletTransaction.AccountNumber

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

	metadata = t.MutateMetadataByAccountEntity(account.Entity, metadata)

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     t.config.AccountConfig.SystemAccountNumber,
			ToAccount:       parentWalletTransaction.AccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "DSBAB",
			OrderType:       "DSB",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     parentWalletTransaction.Description,
			Metadata:        metadata,
		},
	}, nil
}
