package models

import (
	"encoding/json"
	"fmt"
	"time"

	goAcuanLibModel "bitbucket.org/Amartha/go-acuan-lib/model"
	"github.com/shopspring/decimal"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
)

func ConvertTransactionTopUpToReconRecord(trx Transaction) (*ReconRecord, error) {
	var metadata goAcuanLibModel.TopupMetadata

	if trx.Metadata != "" {
		err := json.Unmarshal([]byte(trx.Metadata), &metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	identifier := ""
	if metadata.VaData != nil {
		identifier = metadata.VaData.VirtualAccountNo
	} else {
		identifier = trx.RefNumber
	}

	return &ReconRecord{
		Identifier:   identifier,
		Amount:       trx.Amount.Decimal,
		PaymentDate:  trx.TransactionDate.Format(common.DateFormatDDMMMYYYY),
		RefNumber:    trx.RefNumber,
		CustomerName: "",
		LenderID:     "",
		Status:       StatusReconRecordNotChecked,
	}, nil
}

func ConvertStringCSVToReconRecord(data []string) (*ReconRecord, error) {
	if len(data) != 4 {
		return nil, fmt.Errorf("data length is not 5")
	}

	amount, err := decimal.NewFromString(data[1])
	if err != nil {
		return nil, fmt.Errorf("failed to convert string to decimal: %w", err)
	}

	_, err = time.Parse(common.DateFormatDDMMMYYYY, data[2])
	if err != nil {
		return nil, fmt.Errorf("failed to parse payment date: %w", err)
	}

	return &ReconRecord{
		Identifier:  data[0],
		Amount:      amount,
		PaymentDate: data[2],
		Status:      StatusReconRecordNotChecked,
	}, nil
}
