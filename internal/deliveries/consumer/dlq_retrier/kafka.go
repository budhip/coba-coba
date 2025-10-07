package dlqretrier

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/graceful"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/messaging"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/Shopify/sarama"
	"golang.org/x/sync/errgroup"
)

const logMessage = "[KAFKA-CONSUMER] [DLQ-RETRIER] "

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	ctx       context.Context
	ctxCancel context.CancelFunc

	clientID    string
	cfg         config.Config
	consumerCfg config.ConsumerConfig

	cg sarama.ConsumerGroup

	dlqRetrierHandler   sarama.ConsumerGroupHandler
	dlqProcessorService services.DLQProcessorService

	metrics         metrics.Metrics
	consumerMetrics *metrics.ConsumerMetrics
}

func New(ctx context.Context, cfg config.Config, dp services.DLQProcessorService, metrics metrics.Metrics) (*Consumer, error) {
	c := &Consumer{
		ctx:                 ctx,
		cfg:                 cfg,
		consumerCfg:         cfg.MessageBroker.KafkaConsumer,
		dlqProcessorService: dp,
		metrics:             metrics,
	}

	xlog.Info(c.ctx, logMessage, xlog.String("status", "success init kafka consumer"))

	return c, nil
}

func (c *Consumer) preStart() error {
	saramaCfg, err := messaging.CreateSaramaConsumerConfig(c.consumerCfg, logMessage)
	if err != nil {
		return fmt.Errorf("failed to create consumer config: %w", err)
	}

	if c.consumerCfg.TopicDLQ == "" {
		return errors.New("no DLQ topics given to be consumed, please set the topic")
	}

	if c.consumerCfg.TopicAccountMutationDLQ == "" {
		return errors.New("no account mutation DLQ topics given to be consumed, please set the topic")
	}

	if c.consumerCfg.ConsumerGroupDLQRetrier == "" {
		return errors.New("no kafka DLQ consumer group retrier defined, please set the group")
	}

	// prometheus metrics
	if c.metrics != nil {
		c.consumerMetrics = metrics.NewConsumerMetrics(c.consumerCfg.ConsumerGroupDLQRetrier, c.cfg.App.Name, 1*time.Second, c.metrics.PrometheusRegisterer())
		c.consumerMetrics.Run()
	}

	c.clientID = saramaCfg.ClientID
	c.dlqRetrierHandler = NewRetrierHandler(c.clientID, c.dlqProcessorService, c.consumerCfg, c.consumerMetrics)

	client, err := sarama.NewConsumerGroup(c.consumerCfg.Brokers, c.consumerCfg.ConsumerGroupDLQRetrier, saramaCfg)
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
				if err := c.cg.Consume(ctx, []string{c.consumerCfg.TopicDLQ, c.consumerCfg.TopicAccountMutationDLQ}, c.dlqRetrierHandler); err != nil {
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
