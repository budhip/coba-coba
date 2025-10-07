package transformer

import (
	"context"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/maps"
)

type admfeTransformer struct {
	baseWalletTransactionTransformer
}

func (t admfeTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
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

	entity := t.config.AccountConfig.MapAccountEntity[entityCode]

	receivableAccountNumber, err := getAccountNumberFromConfig(t.config.AccountConfig.OperationalReceivableAccountNumberByEntity, entity)
	if err != nil {
		return nil, err
	}
	transactionType, err := t.masterDataRepository.GetTransactionType(ctx, parentWalletTransaction.TransactionType)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction type: %w with transaction type code: %s", err, parentWalletTransaction.TransactionType)
	}

	metadata := parentWalletTransaction.Metadata
	maps.Copy(metadata, map[string]any{
		"accountNumberBank": t.GenerateAccountNumberBankForMetadataADMFE(parentWalletTransaction),
		"accountNumber":     parentWalletTransaction.AccountNumber,
		"amount":            parentWalletTransaction.NetAmount.ValueDecimal.String(),
		"adminFee":          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
	})
	metadata = t.MutateMetadataByAccountEntity(entityCode, metadata)

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     receivableAccountNumber,
			ToAccount:       t.config.AccountConfig.SystemAccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "ADMFE",
			OrderType:       "ADM",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     fmt.Sprintf("Admin Fee for %s", transactionType.TransactionTypeName),
			Metadata:        metadata,
		},
	}, nil
}
