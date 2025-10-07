package services

import ()

type MoneyFlowCalcService interface {
}

type moneyFlowCalc service

var _ MoneyFlowCalcService = (*moneyFlowCalc)(nil)
