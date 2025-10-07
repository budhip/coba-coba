package transformer

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type dbfacTransformer struct {
	baseWalletTransactionTransformer
}

func (t dbfacTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	disbursementDate, ok := parentWalletTransaction.Metadata["disbursementDate"].(string)
	if !ok || disbursementDate == "" {
		return nil, common.ErrMissingDisbursementDateFromMetadata
	}

	virtualAccountPoint, ok := parentWalletTransaction.Metadata["virtualAccountPoint"].(string)
	if !ok || virtualAccountPoint == "" {
		return nil, common.ErrMissingVirtualAccountPointFromMetadata
	}

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     parentWalletTransaction.AccountNumber,
			ToAccount:       t.config.AccountConfig.BPE,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "DBFAC",
			OrderType:       "DBF",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     parentWalletTransaction.Description,
			Metadata:        parentWalletTransaction.Metadata,
		},
	}, nil
}
