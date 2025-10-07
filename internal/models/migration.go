package models

import (
	"encoding/json"
	"fmt"

	acuanModel "bitbucket.org/Amartha/go-acuan-lib/model"
	"bitbucket.org/Amartha/script-acting-migration/pkg/txtransformer"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
)

type MigrationHeaders = acuanModel.Headers

type MigrationBody struct {
	Data txtransformer.IbuDBTransaction `json:"data"`
}

type PayloadMigrationTransaction struct {
	Headers MigrationHeaders `json:"headers"`
	Body    MigrationBody    `json:"body"`
}

func SerializeTransformedAcuan(acuanTx txtransformer.AcuanTransaction) (req TransactionReq, err error) {
	var metadata map[string]any
	err = json.Unmarshal([]byte(acuanTx.Metadata), &metadata)
	if err != nil {
		return req, fmt.Errorf("failed to serialize metadata: %w", err)
	}

	return TransactionReq{
		TransactionID:   acuanTx.TransactionID,
		FromAccount:     acuanTx.FromAccount,
		ToAccount:       acuanTx.ToAccount,
		TransactionDate: common.FormatDatetimeToStringInLocalTime(acuanTx.TransactionDate, common.DateFormatYYYYMMDD),
		Amount:          acuanTx.Amount,
		Status:          acuanTx.Status,
		Method:          acuanTx.Method,
		TypeTransaction: acuanTx.TypeTransaction,
		Description:     acuanTx.Description,
		RefNumber:       acuanTx.RefNumber,
		Metadata:        metadata,
		OrderTime:       acuanTx.OrderTime,
		OrderType:       acuanTx.OrderType,
		TransactionTime: acuanTx.TransactionTime,
		Currency:        acuanTx.Currency,
	}, nil
}
