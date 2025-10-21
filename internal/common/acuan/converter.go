package acuan

import (
	"fmt"
	"strconv"

	"bitbucket.org/Amartha/go-acuan-lib/model"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func ToAcuanTransaction(
	id string,
	amount decimal.Decimal,
	currency string,
	sourceAccountId string,
	destinationAccountId string,
	description string,
	method string,
	transactionType string,
	transactionTime *model.AcuanTime,
	status string,
	meta interface{},
) (*model.Transaction, error) {
	transactionId, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction id: %v", err)
	}

	rawStatus, err := strconv.ParseUint(status, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse status: %v", err)
	}

	txType := model.TransactionType(transactionType)
	txMethod := model.TransactionMethod(method)
	txStatus := model.TransactionStatus(uint8(rawStatus))

	return &model.Transaction{
		Id:                   &transactionId,
		Amount:               amount,
		Currency:             currency,
		SourceAccountId:      sourceAccountId,
		DestinationAccountId: destinationAccountId,
		Description:          description,
		Method:               txMethod,
		TransactionType:      txType,
		TransactionTime:      *transactionTime,
		Status:               txStatus,
		Meta:                 meta,
	}, nil
}
