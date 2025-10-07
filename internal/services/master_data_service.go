package services

import (
	"context"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"

	"golang.org/x/exp/slices"
)

type MasterDataService interface {
	GetAllOrderType(ctx context.Context, filter models.FilterMasterData) (output []models.OrderType, err error)
	GetOneOrderType(ctx context.Context, orderTypeCode string) (output *models.OrderType, err error)
	GetAllTransactionType(ctx context.Context, filter models.FilterMasterData) (output []models.TransactionType, err error)
	GetOneTransactionType(ctx context.Context, transactionTypeCode string) (output *models.TransactionType, err error)

	CreateOrderType(ctx context.Context, ot models.OrderType) (err error)
	UpdateOrderType(ctx context.Context, ot models.OrderType) (err error)

	GetAllVATConfig(ctx context.Context) (output []models.ConfigVatRevenue, err error)
	UpsertVATConfig(ctx context.Context, configs []models.ConfigVatRevenue) (err error)
}

type masterData service

// GetOneTransactionType implements MasterDataService.
func (m *masterData) GetOneTransactionType(ctx context.Context, transactionTypeCode string) (output *models.TransactionType, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	return m.srv.masterDataRepo.GetTransactionType(ctx, transactionTypeCode)
}

var _ MasterDataService = (*masterData)(nil)

func (m *masterData) GetAllOrderType(ctx context.Context, filter models.FilterMasterData) (output []models.OrderType, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	return m.srv.masterDataRepo.GetListOrderType(ctx, filter)
}

// GetOneOrderType implements MasterDataService.
func (m *masterData) GetOneOrderType(ctx context.Context, orderTypeCode string) (output *models.OrderType, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	return m.srv.masterDataRepo.GetOrderType(ctx, orderTypeCode)
}

func (m *masterData) GetAllTransactionType(ctx context.Context, filter models.FilterMasterData) (output []models.TransactionType, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	return m.srv.masterDataRepo.GetListTransactionType(ctx, filter)
}

func (m *masterData) CreateOrderType(ctx context.Context, ot models.OrderType) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	codes, err := m.srv.masterDataRepo.GetListOrderTypeCode(ctx)
	if err != nil {
		return
	}

	if slices.Contains(codes, ot.OrderTypeCode) {
		return common.ErrDataExist
	}

	return m.srv.masterDataRepo.UpsertOrderType(ctx, ot)
}

func (m *masterData) UpdateOrderType(ctx context.Context, ot models.OrderType) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	codes, err := m.srv.masterDataRepo.GetListOrderTypeCode(ctx)
	if err != nil {
		return
	}

	isCodeNotExists := !slices.Contains(codes, ot.OrderTypeCode)
	if isCodeNotExists {
		return common.ErrDataNotFound
	}

	return m.srv.masterDataRepo.UpsertOrderType(ctx, ot)
}

func (m *masterData) EnsureTransactionTypeExist(ctx context.Context, transactionTypes []string) (err error) {
	masterTrxTypes, err := m.srv.masterDataRepo.GetListTransactionTypeCode(ctx)
	if err != nil {
		err = fmt.Errorf("failed to get transaction type data: %w", err)
		return
	}

	acceptedTransactionType := append(m.srv.conf.TransactionValidationConfig.AcceptedTransactionType, masterTrxTypes...)
	for _, trxType := range transactionTypes {
		if ok := slices.Contains(acceptedTransactionType, trxType); !ok {
			err = fmt.Errorf("invalid transaction type: %s", trxType)
			return
		}
	}

	return
}

func (m *masterData) EnsureOrderTypeExist(ctx context.Context, orderTypes []string) (err error) {
	masterOrderTypes, err := m.srv.masterDataRepo.GetListOrderTypeCode(ctx)
	if err != nil {
		err = fmt.Errorf("failed to get order type data: %w", err)
		return
	}

	acceptedOrderType := append(m.srv.conf.TransactionValidationConfig.AcceptedOrderType, masterOrderTypes...)
	for _, orderType := range orderTypes {
		if ok := slices.Contains(acceptedOrderType, orderType); !ok {
			err = fmt.Errorf("invalid order type: %s", orderType)
			return
		}
	}

	return
}

func (m *masterData) GetAllVATConfig(ctx context.Context) (output []models.ConfigVatRevenue, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	return m.srv.masterDataRepo.GetConfigVATRevenue(ctx)
}

func (m *masterData) UpsertVATConfig(ctx context.Context, configs []models.ConfigVatRevenue) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	return m.srv.masterDataRepo.UpsertConfigVATRevenue(ctx, configs)
}
