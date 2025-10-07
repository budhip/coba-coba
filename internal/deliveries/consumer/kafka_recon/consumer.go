package kafkarecon

import (
	"context"
	"errors"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/config"

	goacuanlib "bitbucket.org/Amartha/go-acuan-lib/model"
	xlog "bitbucket.org/Amartha/go-x/log"
)

type Consumer interface {
	Consume(dateLimit time.Time, initialOffset int64, processor func(transactions []goacuanlib.Transaction))
}

type balanceReconConsumer struct {
	reconConsumer *ReconConsumer
}

// Consume implements Consumer.
func (c *balanceReconConsumer) Consume(dateLimit time.Time, initialOffset int64, processor func(transactions []goacuanlib.Transaction)) {
	c.reconConsumer.dateLimit = dateLimit
	c.reconConsumer.initialOffset = initialOffset
	c.reconConsumer.processor = processor

	c.reconConsumer.Start()
	c.reconConsumer.Stop()
}

func NewBalanceReconConsumer(ctx context.Context, cfg config.Config) (Consumer, error) {
	reconConsumer := &ReconConsumer{
		ctx:         ctx,
		consumerCfg: cfg.MessageBroker.KafkaConsumer,
	}

	if len(reconConsumer.consumerCfg.Brokers) == 0 {
		xlog.Error(context.Background(), "no kafka bootstrap brokers defined, please set the brokers")
		return nil, errors.New("no kafka bootstrap brokers defined, please set the brokers")
	}

	if reconConsumer.consumerCfg.Topic == "" {
		return nil, errors.New("no topics given to be consumed, please set the topic")
	}

	if reconConsumer.consumerCfg.ConsumerGroup == "" {
		return nil, errors.New("no kafka consumer group defined, please set the group")
	}

	xlog.Info(reconConsumer.ctx, logMessage, xlog.String("status", "success init kafka recon consumer"))

	return &balanceReconConsumer{
		reconConsumer: reconConsumer,
	}, nil
}
