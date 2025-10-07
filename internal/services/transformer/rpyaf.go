package transformer

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type rpyafTransformer struct {
	baseWalletTransactionTransformer
}

func (t rpyafTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	configVAT, err := t.masterDataRepository.GetConfigVATRevenue(ctx)
	if err != nil {
		return nil, err
	}

	if parentWalletTransaction.Description == "" {
		return nil, common.ErrMissingDescription
	}

	VATOut, err := t.accountConfigRepository.GetVatOut(
		ctx,
		parentWalletTransaction.AccountNumber,
		parentWalletTransaction.Description)
	if err != nil {
		return nil, err
	}

	amarthaRevenue, err := t.accountConfigRepository.GetRevenue(
		ctx,
		parentWalletTransaction.AccountNumber,
		parentWalletTransaction.Description)
	if err != nil {
		return nil, err
	}

	account, err := t.accountRepository.GetCachedAccount(ctx, amarthaRevenue)
	if err != nil {
		return res, err
	}

	entityCode := account.Entity
	if entityCode == "" {
		return res, common.ErrMissingEntityFromAccount
	}

	metadata := parentWalletTransaction.Metadata
	metadata["loanAccountNumber"] = parentWalletTransaction.AccountNumber
	metadata = t.MutateMetadataByAccountEntity(entityCode, metadata)

	rpyafAmount := amount.ValueDecimal.Decimal

	entity := t.config.AccountConfig.MapAccountEntity[entityCode]

	var rpyagAmount decimal.Decimal
	if entity != "AWF" {
		rpyagAmount, err = calculateVAT(
			amount.ValueDecimal.Decimal,
			parentWalletTransaction.TransactionTime,
			WithVATRevenueConfig(configVAT),
		)
		if err != nil {
			return nil, err
		}
		rpyafAmount = amount.ValueDecimal.Decimal.Sub(rpyagAmount)
	}

	transactions := []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     t.config.AccountConfig.SystemAccountNumber,
			ToAccount:       amarthaRevenue,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(rpyafAmount),
			Status:          string(status),
			TypeTransaction: "RPYAF",
			OrderType:       "RPY",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     parentWalletTransaction.Description,
			Metadata:        metadata,
		},
	}

	if entity != "AWF" {
		transactions = append(transactions, models.TransactionReq{
			TransactionID:   uuid.New().String(),
			FromAccount:     t.config.AccountConfig.SystemAccountNumber,
			ToAccount:       VATOut,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(rpyagAmount),
			Status:          string(status),
			TypeTransaction: "RPYAG",
			OrderType:       "RPY",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     parentWalletTransaction.Description,
			Metadata:        metadata,
		})
	}

	return transactions, nil
}
