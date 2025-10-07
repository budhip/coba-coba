package transformer

import (
	"context"
	"errors"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type rfdtxTransformer struct {
	baseWalletTransactionTransformer
}

func (t rfdtxTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	if parentWalletTransaction.DestinationAccountNumber == "" {
		return nil, common.ErrMissingDestinationAccountNumber
	}

	if parentWalletTransaction.Description == "" {
		return nil, common.ErrMissingDescription
	}

	transaction, err := t.transaction.GetByTransactionID(ctx, parentWalletTransaction.Description)
	if err != nil {
		if errors.Is(err, common.ErrNoRows) {
			return nil, fmt.Errorf("%w: transaction_id not found", common.ErrInvalidRefundData)
		}
		return nil, err
	}

	if transaction.RefNumber != parentWalletTransaction.RefNumber {
		return nil, fmt.Errorf("%w: input refNumber is not match with acuan ref_number column", common.ErrInvalidRefundData)
	}

	if transaction.ToAccount != parentWalletTransaction.AccountNumber {
		return nil, fmt.Errorf("%w: input accountNumber not match with acuan to_account column", common.ErrInvalidRefundData)
	}

	if transaction.FromAccount != parentWalletTransaction.DestinationAccountNumber {
		return nil, fmt.Errorf("%w: input destinationAccountNumber not match with acuan from_account column", common.ErrInvalidRefundData)
	}

	if !transaction.Amount.Decimal.Equal(parentWalletTransaction.NetAmount.ValueDecimal.Decimal) {
		return nil, fmt.Errorf("%w: input netAmount not match with acuan amount column", common.ErrInvalidRefundData)
	}

	if len(parentWalletTransaction.Amounts) > 0 {
		return nil, fmt.Errorf("%w: unsupported RFDTX transaction with multiple child transaction", common.ErrInvalidRefundData)
	}

	return []models.TransactionReq{
		{
			TransactionID:   uuid.New().String(),
			FromAccount:     parentWalletTransaction.AccountNumber,
			ToAccount:       parentWalletTransaction.DestinationAccountNumber,
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(amount.ValueDecimal.Decimal),
			Status:          string(status),
			TypeTransaction: "RFDTX",
			OrderType:       "RFD",
			OrderTime:       getOrderTime(parentWalletTransaction),
			RefNumber:       parentWalletTransaction.RefNumber,
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Description:     parentWalletTransaction.Description,
			Metadata:        parentWalletTransaction.Metadata,
		},
	}, nil
}
