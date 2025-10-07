package moneyflowcalc

import (
	"context"
	"fmt"

	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/graceful"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/messaging"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/Shopify/sarama"
)

const logMessage = "[KAFKA-CONSUMER] [MONEY-FLOW-CALC] "

// Consumer represent a sarama consumer group consumer
type Consumer struct {
	ctx							context.Context

	clientID					string
	cfg							config.Config
	consumerCfg 				config.ConsumerConfig

	cg 							sarama.ConsumerGroup

	MoneyFlowCalcGetterHander	sarama.ConsumerGroupHandler
	
	dlq 						dlqpublisher.Publisher
	mfcs 						services.MoneyFlowCalcService
	metrics 					metrics.Metrics
	consumerMetrics 			*metrics.ConsumerMetrics
}

func New(ctx context.Context, cfg config.Config mfcs services.MoneyFlowCalcService, dlq dlqpublisher.Publisher, metrics metrics.Metrics) (*Consumer, error) {
	c := &Consumer{
		ctx: ctx,
		cfg: cfg,
		consumerCfg: cfg.MessageBroker.KafkaConsumer,
		mfcs: mfcs,
		dlq: dlq,
		metrics: metrics,
	}

	xlog.Info(c.ctx, logMessage, xlog.String("status", "success init kafka consumer"))
	
	return c, nil
}

func (c *Consumer) preStart() error {
	saramaCfg, err := messsaging.CreateSaramaConsumerConfig(c.consumerCfg, logMessage)
	if err != nil {
		xlog.Error(c.ctx, logMessage, xlog.Err(err))
		return fmt.Errorf("failed to create consumer config: %w", err)
	}

	if c.consumerCfg.Topic
}


