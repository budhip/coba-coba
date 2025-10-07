package models

import (
	"encoding/json"
	"fmt"

	goacuanlib "bitbucket.org/Amartha/go-acuan-lib/model"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
)

type StatusTransactionNotification string

const (
	StatusTransactionNotificationSuccess StatusTransactionNotification = "SUCCESS"
	StatusTransactionNotificationFailed  StatusTransactionNotification = "FAILED"
	StatusTransactionNotificationSkipped StatusTransactionNotification = "SKIPPED"
)

type AccountBalanceNotification struct {
	Before Balance `json:"before"`
	After  Balance `json:"after"`
}

func (abn AccountBalanceNotification) MarshalJSON() ([]byte, error) {
	res := struct {
		Before balanceJSONV2 `json:"before"`
		After  balanceJSONV2 `json:"after"`
	}{
		Before: newBalanceJSONV2(abn.Before),
		After:  newBalanceJSONV2(abn.After),
	}

	return json.Marshal(res)
}

type TransactionNotificationPayload struct {
	Identifier        string                                   `json:"identifier"`
	Status            StatusTransactionNotification            `json:"status"`
	WalletTransaction *WalletTransactionResponse               `json:"walletTransaction"`
	AccountBalances   map[string]AccountBalanceNotification    `json:"accountBalances"`
	AcuanData         goacuanlib.Payload[goacuanlib.DataOrder] `json:"acuanData"`
	Message           string                                   `json:"message"`
	ClientID          string                                   `json:"clientID"`
}

func CreateWalletNotificationPayload(
	walletTransaction WalletTransaction,
	acuanTransactions []Transaction,
	beforeBalances map[string]Balance,
	afterBalances map[string]Balance,
	status StatusTransactionNotification,
	message string,
	clientID string,
) (*TransactionNotificationPayload, error) {
	accountBalances := make(map[string]AccountBalanceNotification)

	for accountNumber, beforeBalance := range beforeBalances {
		afterBalance, ok := afterBalances[accountNumber]
		if !ok {
			return nil, fmt.Errorf("account number %s not found in afterBalances", accountNumber)
		}

		accountBalances[accountNumber] = AccountBalanceNotification{
			Before: beforeBalance,
			After:  afterBalance,
		}
	}

	var acuanLibTransactions []goacuanlib.Transaction
	for _, transaction := range acuanTransactions {
		acuanLibTransaction, err := transaction.ToAcuanLibTransaction()
		if err != nil {
			return nil, fmt.Errorf("failed to convert transaction to acuan lib transaction: %w", err)
		}

		acuanLibTransactions = append(acuanLibTransactions, *acuanLibTransaction)
	}

	walletResponse := walletTransaction.ToResponse()

	if len(walletTransaction.TransactionType) < 3 {
		return nil, fmt.Errorf("%w: order type is less than 3 characters %s", common.ErrFailedToCreateNotificationPayload, walletTransaction.TransactionType)
	}

	orderType := walletTransaction.TransactionType[:3]

	return &TransactionNotificationPayload{
		Identifier:        walletTransaction.RefNumber,
		Status:            status,
		WalletTransaction: &walletResponse,
		AccountBalances:   accountBalances,
		AcuanData: goacuanlib.Payload[goacuanlib.DataOrder]{
			Headers: goacuanlib.Headers{
				SourceSystem: "go-fp-transaction",
			},
			Body: goacuanlib.Body[goacuanlib.DataOrder]{
				Data: goacuanlib.DataOrder{
					Order: goacuanlib.Order{
						OrderType: goacuanlib.OrderType(orderType),
						OrderTime: goacuanlib.AcuanTime{
							Time: &walletTransaction.CreatedAt,
						},
						RefNumber:    walletTransaction.RefNumber,
						Transactions: acuanLibTransactions,
					},
				},
			},
		},
		Message:  message,
		ClientID: clientID,
	}, nil
}
