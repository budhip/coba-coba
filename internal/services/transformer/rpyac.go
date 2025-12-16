package transformer

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type rpyacTransformer struct {
	baseWalletTransactionTransformer
}

func (t rpyacTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	if parentWalletTransaction.Description == "" {
		return nil, common.ErrMissingDescription
	}

	accountNumber := parentWalletTransaction.AccountNumber

	useExternalAccountConfig := t.flag.IsEnabled(t.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal)
	if useExternalAccountConfig {
		accountNumber = getLoanAccountNumber(parentWalletTransaction.Metadata)
		if accountNumber == "" {
			return nil, common.ErrMissingLoanAccountNumberFromMetadata
		}
	}

	wht2326, err := t.accountConfigRepository.GetWht2326(ctx, accountNumber, parentWalletTransaction.Description)
	if err != nil {
		return nil, err
	}

	account, err := t.accountRepository.GetCachedAccount(ctx, wht2326)
	if err != nil {
		return res, err
	}

	entityCode := account.Entity
	if entityCode == "" {
		return res, common.ErrMissingEntityFromAccount
	}

	metadata := parentWalletTransaction.Metadata
	metadata = t.MutateMetadataByAccountEntity(entityCode, metadata)

	fromAccount := t.config.AccountConfig.SystemAccountNumber

	// Check feature flag for RPYAB + RPYAC adjustment
	isNeedJogressRpyabRpyac := t.config.FeatureFlag.EnableRpyabRpyacAdjustment
	if isNeedJogressRpyabRpyac {
		// If feature flag is ON, switch fromAccount to account number
		fromAccount = parentWalletTransaction.AccountNumber
	}

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     fromAccount,
			ToAccount:       wht2326,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "RPYAC",
			OrderType:       "RPY",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     parentWalletTransaction.Description,
			Metadata:        metadata,
		},
	}, nil
}
