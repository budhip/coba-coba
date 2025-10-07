package transformer

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/slices"
)

type dsbrpTransformer struct {
	baseWalletTransactionTransformer
}

func (t dsbrpTransformer) createDefaultDSB(
	ctx context.Context,
	fromAccount, toAccount string,
	amount decimal.Decimal,
	typeTransaction string,
	parentWalletTransaction models.WalletTransaction,
) (res models.TransactionReq, err error) {
	status, err := transformWalletTransactionStatus(parentWalletTransaction.Status)
	if err != nil {
		return
	}

	account, err := t.accountRepository.GetCachedAccount(ctx, toAccount)
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

	return models.TransactionReq{
		TransactionID:   uuid.New().String(),
		FromAccount:     fromAccount,
		ToAccount:       toAccount,
		TransactionDate: common.FormatDatetimeToStringInLocalTime(parentWalletTransaction.TransactionTime, common.DateFormatYYYYMMDD),
		Amount:          decimal.NewNullDecimal(amount),
		Status:          string(status),
		TypeTransaction: typeTransaction,
		OrderType:       "DSB",
		OrderTime:       getOrderTime(parentWalletTransaction),
		RefNumber:       parentWalletTransaction.RefNumber,
		Currency:        transformCurrency(parentWalletTransaction.NetAmount.Currency),
		TransactionTime: parentWalletTransaction.TransactionTime,
		Description:     parentWalletTransaction.Description,
		Metadata:        metadata,
	}, nil
}

func (t dsbrpTransformer) getRevenueAccountNumber(ctx context.Context, parentWalletTransaction models.WalletTransaction) (string, error) {
	if parentWalletTransaction.Description == "MODAL_LOAN" {
		account, err := t.accountRepository.GetCachedAccount(
			ctx,
			parentWalletTransaction.AccountNumber,
		)
		if err != nil {
			return "", err
		}

		entity := t.config.AccountConfig.MapAccountEntity[account.Entity]
		if entity == "" {
			return "", common.ErrMissingEntityFromAccount
		}

		amarthaRevenue, err := getAccountNumberFromConfig(
			t.config.AccountConfig.AmarthaRevenueModalLoanPlatformFeeByEntityCode,
			entity)
		if err != nil {
			return "", err
		}

		return amarthaRevenue, nil
	}

	amarthaRevenue, err := t.accountConfigRepository.GetAdminFee(
		ctx,
		parentWalletTransaction.AccountNumber,
		parentWalletTransaction.Description,
	)
	if err != nil {
		return "", err
	}

	return amarthaRevenue, nil
}

func (t dsbrpTransformer) Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error) {
	amarthaRevenue, err := t.getRevenueAccountNumber(ctx, parentWalletTransaction)
	if err != nil {
		return nil, err
	}

	excludedDSBRQDescription := []string{
		"NORMAL",
		"MODAL_LOAN",
	}
	excludeDSBRQ := slices.Contains(excludedDSBRQDescription, parentWalletTransaction.Description)
	if excludeDSBRQ {
		txDSBRP, err := t.createDefaultDSB(
			ctx,
			t.config.AccountConfig.SystemAccountNumber,
			amarthaRevenue,
			amount.ValueDecimal.Decimal,
			"DSBRP",
			parentWalletTransaction,
		)
		if err != nil {
			return nil, err
		}

		res = append(res, txDSBRP)
	} else {
		VATOut, err := t.accountConfigRepository.GetVatOut(
			ctx,
			parentWalletTransaction.AccountNumber,
			parentWalletTransaction.Description,
		)
		if err != nil {
			return nil, err
		}

		configVAT, err := t.masterDataRepository.GetConfigVATRevenue(ctx)
		if err != nil {
			return nil, err
		}

		amountDSBRQ, err := calculateVAT(
			amount.ValueDecimal.Decimal,
			parentWalletTransaction.TransactionTime,
			WithVATRevenueConfig(configVAT),
		)
		if err != nil {
			return nil, err
		}

		amountDSBRP := amount.ValueDecimal.Sub(amountDSBRQ)

		txDSBRQ, err := t.createDefaultDSB(
			ctx,
			t.config.AccountConfig.SystemAccountNumber,
			VATOut,
			amountDSBRQ,
			"DSBRQ",
			parentWalletTransaction,
		)
		if err != nil {
			return nil, err
		}

		txDSBRP, err := t.createDefaultDSB(
			ctx,
			t.config.AccountConfig.SystemAccountNumber,
			amarthaRevenue,
			amountDSBRP,
			"DSBRP",
			parentWalletTransaction,
		)
		if err != nil {
			return nil, err
		}

		res = append(res, txDSBRQ, txDSBRP)
	}

	return res, nil
}
