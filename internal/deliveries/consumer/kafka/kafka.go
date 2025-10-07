package kafkaconsumer

import (
	"context"
	"errors"
	"fmt"
	"time"

	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/graceful"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/messaging"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/retry"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/transaction_notification"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/Shopify/sarama"
	"golang.org/x/sync/errgroup"
)

const logMessage = "[KAFKA-CONSUMER] [TRANSACTION] "

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	ctx context.Context

	clientID    string
	cfg         config.Config
	consumerCfg config.ConsumerConfig

	cg sarama.ConsumerGroup

	dlq                     dlqpublisher.Publisher
	journal                 JournalPublisher
	transactionNotification transaction_notification.TransactionNotificationPublisher

	trxHandler     sarama.ConsumerGroupHandler
	trxService     services.TransactionService
	accountService services.AccountService

	metrics         metrics.Metrics
	consumerMetrics *metrics.ConsumerMetrics
}

func New(
	ctx context.Context,
	cfg config.Config,
	ts services.TransactionService,
	dlq dlqpublisher.Publisher,
	journal JournalPublisher,
	accSvc services.AccountService,
	transactionNotification transaction_notification.TransactionNotificationPublisher,
	metrics metrics.Metrics) (*Consumer, error) {
	c := &Consumer{
		ctx:                     ctx,
		cfg:                     cfg,
		consumerCfg:             cfg.MessageBroker.KafkaConsumer,
		trxService:              ts,
		dlq:                     dlq,
		journal:                 journal,
		accountService:          accSvc,
		transactionNotification: transactionNotification,
		metrics:                 metrics,
	}

	xlog.Info(c.ctx, logMessage, xlog.String("status", "success init kafka consumer"))

	return c, nil
}

func (c *Consumer) preStart() error {
	saramaCfg, err := messaging.CreateSaramaConsumerConfig(c.consumerCfg, logMessage)
	if err != nil {
		xlog.Error(c.ctx, logMessage, xlog.Err(err))
		return fmt.Errorf("failed to create consumer config: %w", err)
	}

	if c.consumerCfg.Topic == "" {
		return errors.New("no topics given to be consumed, please set the topic")
	}

	if c.consumerCfg.ConsumerGroup == "" {
		return errors.New("no kafka consumer group defined, please set the group")
	}

	// prometheus metrics
	if c.metrics != nil {
		c.consumerMetrics = metrics.NewConsumerMetrics(c.consumerCfg.ConsumerGroup, c.cfg.App.Name, 1*time.Second, c.metrics.PrometheusRegisterer())
		c.consumerMetrics.Run()
	}

	c.clientID = saramaCfg.ClientID

	ebRetryer := retry.NewExponentialBackOff(&c.cfg.ExponentialBackoff)
	c.trxHandler = NewTransactionHandler(
		c.clientID,
		c.trxService,
		c.dlq, ebRetryer,
		c.journal,
		c.accountService,
		c.cfg.FeatureFlag,
		c.transactionNotification,
		c.consumerMetrics)

	client, err := sarama.NewConsumerGroup(c.consumerCfg.Brokers, c.consumerCfg.ConsumerGroup, saramaCfg)
	if err != nil {
		return err
	}
	c.cg = client

	return nil
}

func (c *Consumer) Start() graceful.ProcessStarter {
	return func() error {
		err := c.preStart()
		if err != nil {
			return err
		}

		// track errors
		go func() {
			for errCg := range c.cg.Errors() {
				xlog.Error(c.ctx, logMessage, xlog.Err(fmt.Errorf("client error: %v", xlog.Err(errCg))))
			}
		}()

		eg, ctx := errgroup.WithContext(c.ctx)

		eg.Go(func() error {
			for {
				if err := c.cg.Consume(ctx, []string{c.consumerCfg.Topic}, c.trxHandler); err != nil {
					xlog.Warn(c.ctx, logMessage, xlog.Err(fmt.Errorf("error start consumer: %v", xlog.Err(err))))
				}
				if err := c.ctx.Err(); err != nil {
					return fmt.Errorf("context was canceled: %w", err)
				}
			}
		})

		return eg.Wait()
	}
}

func (c *Consumer) Stop() graceful.ProcessStopper {
	return func(ctx context.Context) error {
		if err := c.cg.Close(); err != nil {
			return err
		}

		return nil
	}
}
