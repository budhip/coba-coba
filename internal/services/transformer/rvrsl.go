package transformer

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
)

type rvrslTransformer struct {
	baseWalletTransactionTransformer
}

func (t rvrslTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return nil, err
	}

	if parentWalletTransaction.TransactionFlow != models.TransactionFlowRefund {
		return nil, common.ErrUnsupportedTransactionFlow
	}

	walletTrxId, ok := parentWalletTransaction.Metadata["walletTransactionId"].(string)
	if !ok || walletTrxId == "" {
		return nil, common.ErrMissingWalletTransactionIdFromMetadata
	}

	limit := t.config.TransactionConfig.ReversalTimeRangeDays
	if limit <= 0 {
		limit = models.DefaultReversalTimeRangeDay
	}

	now, _ := common.NowZeroTime()
	startDate := now.AddDate(0, 0, -limit)

	ts, err := t.transaction.GetList(ctx, models.TransactionFilterOptions{
		Search:    parentWalletTransaction.RefNumber,
		SearchBy:  "refNumber",
		StartDate: &startDate,
		EndDate:   &now,
	})
	if err != nil {
		return nil, err
	}

	if len(ts) == 0 {
		return nil, common.ErrrefNumberNotFound
	}

	for _, transaction := range ts {
		res = append(res, models.TransactionReq{
			RefNumber:       parentWalletTransaction.RefNumber,
			OrderType:       "RVR",
			TypeTransaction: "RVRSL",
			Description:     transaction.TypeTransaction + " " + transaction.TransactionID,
			Amount:          transaction.Amount,
			FromAccount:     transaction.ToAccount,
			ToAccount:       transaction.FromAccount,
			TransactionID:   uuid.New().String(),
			TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
			Status:          string(status),
			OrderTime:       getOrderTime(parentWalletTransaction),
			Currency:        transformCurrency(amount.Currency),
			TransactionTime: parentWalletTransaction.TransactionTime,
			Metadata:        parentWalletTransaction.Metadata,
		})
	}
	return res, nil
}
