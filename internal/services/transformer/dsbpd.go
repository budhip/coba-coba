package transformer

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
)

type dsbpdTransformer struct {
	baseWalletTransactionTransformer
}

func (t dsbpdTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	oldLoanAccountNumber, ok := parentWalletTransaction.Metadata["oldLoanAccountNumber"].(string)
	if !ok || oldLoanAccountNumber == "" {
		return nil, common.ErrMissingOldLoanAccountNumberFromMetadata
	}

	newLoanAccountNumber, ok := parentWalletTransaction.Metadata["newLoanAccountNumber"].(string)
	if !ok || newLoanAccountNumber == "" {
		return nil, common.ErrMissingNewLoanAccountNumberFromMetadata
	}

	oldAccount, err := t.accountRepository.GetCachedAccount(ctx, oldLoanAccountNumber)
	if err != nil {
		return nil, err
	}

	newAccount, err := t.accountRepository.GetCachedAccount(ctx, newLoanAccountNumber)
	if err != nil {
		return nil, err
	}

	newEntityLoanAccountNumber, ok := t.config.AccountConfig.MapAccountEntity[newAccount.Entity]
	if !ok || newEntityLoanAccountNumber == "" {
		return nil, common.ErrNewEntityLoanAccountNumberFromMetadataNotFound
	}

	oldEntityLoanAccountNumber, ok := t.config.AccountConfig.MapAccountEntity[oldAccount.Entity]
	if !ok || oldEntityLoanAccountNumber == "" {
		return nil, common.ErrOldEntityLoanAccountNumberFromMetadataNotFound
	}

	parentWalletTransaction.Metadata["newEntityLoanAccountNumber"] = newAccount.Entity
	parentWalletTransaction.Metadata["oldEntityLoanAccountNumber"] = oldAccount.Entity

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     t.config.AccountConfig.SystemAccountNumber,
			ToAccount:       t.config.AccountConfig.SystemAccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "DSBPD",
			OrderType:       "DSB",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     parentWalletTransaction.Description,
			Metadata:        parentWalletTransaction.Metadata,
		},
	}, nil
}
