package transformer

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type dsbpiTransformer struct {
	baseWalletTransactionTransformer
}

func (t dsbpiTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	account, err := t.accountRepository.GetCachedAccount(ctx, parentWalletTransaction.AccountNumber)
	if err != nil {
		return nil, err
	}

	entityCode := account.Entity
	if entityCode == "" {
		return nil, common.ErrMissingEntityFromAccount
	}

	entity := t.config.AccountConfig.MapAccountEntity[entityCode]

	toAccount, err := getAccountNumberFromConfig(
		t.config.AccountConfig.AccountNumberInsurancePremiumDisbursementByEntity,
		entity,
	)
	if err != nil {
		return nil, err
	}

	metadata := parentWalletTransaction.Metadata
	metadata = t.MutateMetadataByAccountEntity(entityCode, metadata)

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     t.config.AccountConfig.SystemAccountNumber,
			ToAccount:       toAccount,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "DSBPI",
			OrderType:       "DSB",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     "Premi Insurance from Disbursement Loan " + parentWalletTransaction.AccountNumber,
			Metadata:        metadata,
		},
	}, nil
}
