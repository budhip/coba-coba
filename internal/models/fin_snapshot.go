package models

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	LabelOutstanding = "outstanding"
	LabelPrincipal   = "principal"
	LabelAmartha     = "amartha"
	LabelLender      = "lender"
	LabelPPN         = "ppn"
	LabelPPh         = "pph"
)

type CollectRepayment struct {
	TransactionDate time.Time           `json:"transactionDate"`
	Outstanding     decimal.NullDecimal `json:"outstanding"`
	Principal       decimal.NullDecimal `json:"principal"`
	Amartha         decimal.NullDecimal `json:"amartha"`
	Lender          decimal.NullDecimal `json:"lender"`
	PPN             decimal.NullDecimal `json:"ppn"`
	PPh             decimal.NullDecimal `json:"pph"`
}

type DoGetFinSnapshotResponse struct {
	Data []FinSnapshotResponse `json:"data"`
}

type labelList struct {
	LabelA string `json:"labelA,omitempty"`
	LabelB string `json:"labelB,omitempty"`
	LabelC string `json:"labelC,omitempty"`
	LabelD string `json:"labelD,omitempty"`
	LabelE string `json:"labelE,omitempty"`
}

type FinSnapshotResponse struct {
	Timestamp   string  `json:"timestamp"`
	Value       float64 `json:"value"`
	ProcessName string  `json:"processName"`
	Namespace   string  `json:"namespace"`
	labelList
}

func (r CollectRepayment) MapToFinSnapshot() DoGetFinSnapshotResponse {
	namespace := "GO_FP_TRANSACTION_API"
	processName := "collect_repayment_report"
	timestamp := time.Now().Format(time.RFC3339)

	metrics := []struct {
		label string
		value decimal.NullDecimal
	}{
		{LabelOutstanding, r.Outstanding},
		{LabelPrincipal, r.Principal},
		{LabelAmartha, r.Amartha},
		{LabelLender, r.Lender},
		{LabelPPN, r.PPN},
		{LabelPPh, r.PPh},
	}

	var responses []FinSnapshotResponse
	for _, m := range metrics {
		responses = append(responses, FinSnapshotResponse{
			Timestamp:   timestamp,
			Value:       m.value.Decimal.InexactFloat64(),
			ProcessName: processName,
			Namespace:   namespace,
			labelList:   labelList{LabelA: m.label},
		})
	}

	return DoGetFinSnapshotResponse{Data: responses}
}
