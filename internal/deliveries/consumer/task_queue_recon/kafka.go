package queuerecon

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

const logMessage = "[KAFKA-CONSUMER] [TASK-QUEUE-RECON] "

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	ctx context.Context

	clientID    string
	cfg         config.Config
	consumerCfg config.ConsumerConfig

	cg sarama.ConsumerGroup

	TaskQueueReconHandler sarama.ConsumerGroupHandler

	rs services.ReconService

	metrics metrics.Metrics
}

func New(ctx context.Context, cfg config.Config, rs services.ReconService, metrics metrics.Metrics) (*Consumer, error) {
	c := &Consumer{
		ctx:         ctx,
		cfg:         cfg,
		consumerCfg: cfg.MessageBroker.KafkaConsumer,
		rs:          rs,
		metrics:     metrics,
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

	if c.consumerCfg.TopicRecon == "" {
		return errors.New("no topics given to be consumed, please set the topic")
	}

	if c.consumerCfg.ConsumerGroupTaskQueueRecon == "" {
		return errors.New("no kafka consumer group defined, please set the group")
	}

	// prometheus metrics
	if c.metrics != nil {
		saramaCfg.MetricRegistry = c.metrics.SaramaRegistry(c.consumerCfg.ConsumerGroupTaskQueueRecon, 1*time.Second)
	}

	c.clientID = saramaCfg.ClientID
	c.TaskQueueReconHandler = NewTaskQueueReconHandler(c.clientID, c.rs)

	client, err := sarama.NewConsumerGroup(c.consumerCfg.Brokers, c.consumerCfg.ConsumerGroupTaskQueueRecon, saramaCfg)
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
				if err := c.cg.Consume(ctx, []string{c.consumerCfg.TopicRecon}, c.TaskQueueReconHandler); err != nil {
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
