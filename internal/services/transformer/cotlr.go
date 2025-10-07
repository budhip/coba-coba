package transformer

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/maps"
)

type cotlrTransformer struct {
	baseWalletTransactionTransformer
}

func (t cotlrTransformer) Transform(_ context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	metadata := parentWalletTransaction.Metadata
	for _, detail := range parentWalletTransaction.Amounts {
		isAdminFee := detail.Type == "ADMFE"
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

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     parentWalletTransaction.AccountNumber,
			ToAccount:       t.config.AccountConfig.SystemAccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "COTLR",
			OrderType:       "COT",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     parentWalletTransaction.Description,
			Metadata:        metadata,
		},
	}, nil
}
