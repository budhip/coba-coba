package acuanclient

import (
	"context"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/config"

	goAcuanLib "bitbucket.org/Amartha/go-acuan-lib"
	goAcuanLibModel "bitbucket.org/Amartha/go-acuan-lib/model"
	xlog "bitbucket.org/Amartha/go-x/log"
)

type AcuanClient interface {
	PublishTransaction(ctx context.Context, req PublishTransactionRequest) (err error)
}

type client struct {
	acuanClient *goAcuanLib.AcuanLib
}

func New(cfg config.Config) (AcuanClient, error) {
	acuanConfig := &goAcuanLib.Config{
		Kafka: &goAcuanLib.KafkaConfig{
			BrokerList:        cfg.AcuanLibConfig.Kafka.BrokerList,
			PartitionStrategy: cfg.AcuanLibConfig.Kafka.PartitionStrategy,
		},
		SourceSystem:          cfg.AcuanLibConfig.SourceSystem,
		Topic:                 cfg.AcuanLibConfig.Topic,
		TopicAccounting:       cfg.AcuanLibConfig.TopicAccounting,
		TopUpKey:              cfg.AcuanLibConfig.TopUpKey,
		InvestmentKey:         cfg.AcuanLibConfig.InvestmentKey,
		CashoutKey:            cfg.AcuanLibConfig.CashoutKey,
		DisbursementKey:       cfg.AcuanLibConfig.DisbursementKey,
		DisbursementFailedKey: cfg.AcuanLibConfig.DisbursementFailedKey,
		RepaymentKey:          cfg.AcuanLibConfig.RepaymentKey,
		RefundKey:             cfg.AcuanLibConfig.RefundKey,
	}

	if acuanConfig.SourceSystem == "" {
		acuanConfig.SourceSystem = cfg.App.Name
	}

	acuanClient, err := goAcuanLib.NewClient(acuanConfig)
	if err != nil {
		return nil, fmt.Errorf("failed connect to acuan client: %v", err)
	}
	return &client{acuanClient}, nil
}

func (c *client) PublishTransaction(ctx context.Context, req PublishTransactionRequest) (err error) {
	currency := goAcuanLibModel.TransactionCurrencyIDR
	if req.Currency != "" {
		currency = req.Currency
	}

	err = c.acuanClient.General.PublishOrder(
		goAcuanLibModel.OrderType(req.OrderType),
		req.RefNumber,
		[]goAcuanLibModel.Transaction{
			{
				Amount:               req.Amount,
				Currency:             currency,
				SourceAccountId:      req.FromAccount,
				DestinationAccountId: req.ToAccount,
				Description:          req.Description,
				Method:               req.Method,
				TransactionType:      req.TransactionType,
				TransactionTime:      goAcuanLibModel.AcuanTime{Time: &req.TransactionTime},
				Status:               goAcuanLibModel.TransactionStatusSuccess,
				Meta:                 req.Metadata,
			},
		})
	if err != nil {
		xlog.Warn(ctx, "PublishTransaction", xlog.String("status", "fail"), xlog.Any("message", req), xlog.Err(err))
		return
	}
	xlog.Info(ctx, "PublishTransaction", xlog.String("status", "success"), xlog.Any("message", req))

	return
}
