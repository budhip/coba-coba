package transformer

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// tupviTransformer is a transformer to acuan transaction for TUPVI transaction type
type tupviTransformer struct {
	baseWalletTransactionTransformer
}

func (t tupviTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	account, err := t.accountRepository.GetCachedAccount(
		ctx,
		parentWalletTransaction.AccountNumber,
	)
	if err != nil {
		return nil, err
	}

	entityCode := account.Entity
	if entityCode == "" {
		return nil, common.ErrMissingEntityFromAccount
	}

	parentWalletTransaction.Metadata["entity"] = t.config.AccountConfig.MapAccountEntity[entityCode]
	metadata := parentWalletTransaction.Metadata
	for _, detail := range parentWalletTransaction.Amounts {
		isAdminFee := slices.Contains([]string{"TUPVI", "ADMFE"}, detail.Type)
		if isAdminFee {
			adminFee := detail.Amount.ValueDecimal
			maps.Copy(metadata, map[string]any{
				"accountNumberBank": t.GenerateAccountNumberBankForMetadataADMFE(parentWalletTransaction),
				"accountNumber":     parentWalletTransaction.AccountNumber,
				"amount":            amount.ValueDecimal.String(),
				"adminFee":          adminFee.String(),
				"net":               amount.ValueDecimal.Sub(adminFee.Decimal).String(),
			})
			break
		}
	}

	metadata = t.MutateMetadataByAccountEntity(entityCode, metadata)

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     t.config.AccountConfig.SystemAccountNumber,
			ToAccount:       parentWalletTransaction.AccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "TUPVI",
			OrderType:       "TUP",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     parentWalletTransaction.Description,
			Metadata:        metadata,
		},
	}, nil
}
