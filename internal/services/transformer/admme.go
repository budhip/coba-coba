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

type admmeTransformer struct {
	baseWalletTransactionTransformer
}

func (t admmeTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	entity := getEntityFromMetadata(parentWalletTransaction.Metadata)
	if entity == "" {
		return nil, common.ErrMissingEntityFromMetadata
	}

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
		"accountNumber": parentWalletTransaction.AccountNumber,
		"amount":        parentWalletTransaction.NetAmount.ValueDecimal.String(),
		"adminFee":      decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
	})

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     receivableAccountNumber,
			ToAccount:       t.config.AccountConfig.SystemAccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "ADMME",
			OrderType:       "ADM",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     fmt.Sprintf("Admin Fee for %s - %s", transactionType.TransactionTypeName, parentWalletTransaction.AccountNumber),
			Metadata:        parentWalletTransaction.Metadata,
		},
	}, nil
}
