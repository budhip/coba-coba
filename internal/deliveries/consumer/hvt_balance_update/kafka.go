package hvtbalanceupdate

import (
	"context"
	"errors"
	"fmt"
	"time"

	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/graceful"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/messaging"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/Shopify/sarama"
	"golang.org/x/sync/errgroup"
)

const logMessage = "[KAFKA-CONSUMER] [HVT-BALANCE-UPDATE]"

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	ctx context.Context
	// ctxCancel context.CancelFunc

	clientID    string
	cfg         config.Config
	consumerCfg config.ConsumerConfig

	cg sarama.ConsumerGroup

	dlq                     dlqpublisher.Publisher
	hvtbalanceupdateHandler sarama.ConsumerGroupHandler
	balanceService          services.BalanceService
	cacheRepo               repositories.CacheRepository

	metrics         metrics.Metrics
	consumerMetrics *metrics.ConsumerMetrics
}

func New(ctx context.Context, cfg config.Config, bs services.BalanceService, metrics metrics.Metrics, cacheRepo repositories.CacheRepository, dlq dlqpublisher.Publisher) (*Consumer, error) {
	c := &Consumer{
		ctx:            ctx,
		cfg:            cfg,
		consumerCfg:    cfg.MessageBroker.KafkaConsumer,
		balanceService: bs,
		metrics:        metrics,
		cacheRepo:      cacheRepo,
		dlq:            dlq,
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

	if c.consumerCfg.TopicBalanceHVT == "" {
		return errors.New("no topics given to be consumed, please set the topic")
	}

	if c.consumerCfg.ConsumerGroupBalanceHvt == "" {
		return errors.New("no kafka consumer group defined, please set the group")
	}

	// prometheus metrics
	if c.metrics != nil {
		c.consumerMetrics = metrics.NewConsumerMetrics(c.consumerCfg.ConsumerGroup, c.cfg.App.Name, 1*time.Second, c.metrics.PrometheusRegisterer())
		c.consumerMetrics.Run()
	}

	c.clientID = saramaCfg.ClientID

	c.hvtbalanceupdateHandler = NewHvtBalanceHandler(
		c.clientID,
		c.balanceService,
		c.cacheRepo,
		c.dlq,
		c.cfg.FeatureFlag,
		c.consumerMetrics,
	)

	client, err := sarama.NewConsumerGroup(c.consumerCfg.Brokers, c.consumerCfg.ConsumerGroupBalanceHvt, saramaCfg)
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
				if err := c.cg.Consume(ctx, []string{c.consumerCfg.TopicBalanceHVT}, c.hvtbalanceupdateHandler); err != nil {
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
