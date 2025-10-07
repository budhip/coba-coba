package models

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type OrderType struct {
	OrderTypeCode    string            `json:"orderTypeCode"`
	OrderTypeName    string            `json:"orderTypeName"`
	TransactionTypes []TransactionType `json:"transactionTypes"`
}

func MakeOrderTypesMap(orderTypes []OrderType) (mapOrderType, mapTransactionType map[string]string) {
	mapOrderType = make(map[string]string, len(orderTypes))
	mapTransactionType = make(map[string]string)

	for _, o := range orderTypes {
		mapOrderType[o.OrderTypeCode] = o.OrderTypeName

		for _, t := range o.TransactionTypes {
			mapTransactionType[t.TransactionTypeCode] = t.TransactionTypeName
		}
	}

	return
}

type OrderTypeOut struct {
	Kind             string               `json:"kind"`
	OrderTypeCode    string               `json:"orderTypeCode"`
	OrderTypeName    string               `json:"orderTypeName"`
	TransactionTypes []TransactionTypeOut `json:"transactionTypes"`
}

func (m OrderType) ToResponse() OrderTypeOut {
	var tts []TransactionTypeOut
	for _, tt := range m.TransactionTypes {
		tts = append(tts, tt.ToResponse())
	}

	return OrderTypeOut{
		Kind:             "orderType",
		OrderTypeCode:    m.OrderTypeCode,
		OrderTypeName:    m.OrderTypeName,
		TransactionTypes: tts,
	}
}

type TransactionType struct {
	TransactionTypeCode string `json:"transactionTypeCode"`
	TransactionTypeName string `json:"transactionTypeName"`
}

type TransactionTypeOut struct {
	Kind                string `json:"kind"`
	TransactionTypeCode string `json:"transactionTypeCode"`
	TransactionTypeName string `json:"transactionTypeName"`
}

func (t TransactionType) ToResponse() TransactionTypeOut {
	return TransactionTypeOut{
		Kind:                "transactionType",
		TransactionTypeCode: t.TransactionTypeCode,
		TransactionTypeName: t.TransactionTypeName,
	}
}

type FilterMasterData struct {
	Name string `query:"name" example:"Top Up Lender P2P"`
	Code string `query:"code" example:"100"`
}

type ConfigVatRevenue struct {
	Percentage decimal.Decimal `json:"percentage"`
	StartTime  time.Time       `json:"startTime"`
	EndTime    time.Time       `json:"endTime"`
}

type ConfigVATRevenueManager struct {
	Config []ConfigVatRevenue
}

func (m ConfigVatRevenue) ToResponse() ConfigVatRevenueOut {
	return ConfigVatRevenueOut{
		Kind:       "configVatRevenue",
		Percentage: m.Percentage,
		StartTime:  m.StartTime,
		EndTime:    m.EndTime,
	}
}

func (m ConfigVATRevenueManager) GetActiveConfig(t time.Time) (*ConfigVatRevenue, error) {
	for _, c := range m.Config {
		if t.After(c.StartTime) && t.Before(c.EndTime) {
			return &c, nil
		}
	}

	return nil, fmt.Errorf("no valid active config found for %v", t)
}

type ConfigVatRevenueOut struct {
	Kind       string          `json:"kind"`
	Percentage decimal.Decimal `json:"percentage"`
	StartTime  time.Time       `json:"startTime"`
	EndTime    time.Time       `json:"endTime"`
}
