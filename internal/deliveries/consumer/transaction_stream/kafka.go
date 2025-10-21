package transaction_stream

import (
	"context"

	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/graceful"
	kafkacommon "bitbucket.org/Amartha/go-fp-transaction/internal/common/kafka"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
	xlog "bitbucket.org/Amartha/go-x/log"
)

const logMessage = "[KAFKA-CONSUMER] [TRANSACTION-STREAM] "

type Consumer struct {
	*kafkacommon.BaseConsumer
	mfs services.MoneyFlowService
	dlq dlqpublisher.Publisher
}

func New(ctx context.Context, cfg config.Config, mfs services.MoneyFlowService, dlq dlqpublisher.Publisher, metrics metrics.Metrics) (*Consumer, error) {
	c := &Consumer{
		mfs: mfs,
		dlq: dlq,
	}

	handler := NewTransactionStreamHandler("", mfs, dlq, cfg, nil)

	baseConsumer, err := kafkacommon.NewBaseConsumer(kafkacommon.BaseConsumerConfig{
		Ctx:           ctx,
		Config:        cfg,
		Metrics:       metrics,
		Handler:       handler,
		LogPrefix:     logMessage,
		Topic:         cfg.MessageBroker.KafkaConsumer.TopicTransactionStream,
		ConsumerGroup: cfg.MessageBroker.KafkaConsumer.ConsumerGroupTransactionStream,
	})
	if err != nil {
		return nil, err
	}

	c.BaseConsumer = baseConsumer

	xlog.Info(ctx, logMessage, xlog.String("status", "success init kafka consumer"))

	return c, nil
}

func (c *Consumer) Start() graceful.ProcessStarter {
	return c.BaseConsumer.Start()
}

func (c *Consumer) Stop() graceful.ProcessStopper {
	return c.BaseConsumer.Stop()
}
