package kafka

import (
	"context"
	"errors"
	"fmt"
	"time"

	xlog "bitbucket.org/Amartha/go-x/log"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/graceful"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/messaging"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"

	"github.com/Shopify/sarama"
	"golang.org/x/sync/errgroup"
)

type BaseConsumer struct {
	ctx             context.Context
	clientID        string
	cfg             config.Config
	consumerCfg     config.ConsumerConfig
	cg              sarama.ConsumerGroup
	handler         sarama.ConsumerGroupHandler
	metrics         metrics.Metrics
	consumerMetrics *metrics.ConsumerMetrics
	logPrefix       string
	topic           string
	consumerGroup   string
}

type BaseConsumerConfig struct {
	Ctx           context.Context
	Config        config.Config
	Metrics       metrics.Metrics
	Handler       sarama.ConsumerGroupHandler
	LogPrefix     string
	Topic         string
	ConsumerGroup string
}

func NewBaseConsumer(cfg BaseConsumerConfig) (*BaseConsumer, error) {
	return &BaseConsumer{
		ctx:           cfg.Ctx,
		cfg:           cfg.Config,
		consumerCfg:   cfg.Config.MessageBroker.KafkaConsumer,
		handler:       cfg.Handler,
		metrics:       cfg.Metrics,
		logPrefix:     cfg.LogPrefix,
		topic:         cfg.Topic,
		consumerGroup: cfg.ConsumerGroup,
	}, nil
}

func (c *BaseConsumer) PreStart() error {
	saramaCfg, err := messaging.CreateSaramaConsumerConfig(c.consumerCfg, c.logPrefix)
	if err != nil {
		xlog.Error(c.ctx, c.logPrefix, xlog.Err(err))
		return fmt.Errorf("failed to create consumer config: %w", err)
	}

	if c.topic == "" {
		return errors.New("no topics given to be consumed, please set the topic")
	}

	if c.consumerGroup == "" {
		return errors.New("no kafka consumer group defined, please set the group")
	}

	if c.metrics != nil {
		c.consumerMetrics = metrics.NewConsumerMetrics(c.consumerGroup, c.cfg.App.Name, 1*time.Second, c.metrics.PrometheusRegisterer())
		c.consumerMetrics.Run()
	}

	c.clientID = saramaCfg.ClientID

	client, err := sarama.NewConsumerGroup(c.consumerCfg.Brokers, c.consumerGroup, saramaCfg)
	if err != nil {
		return err
	}
	c.cg = client

	return nil
}

func (c *BaseConsumer) Start() graceful.ProcessStarter {
	return func() error {
		err := c.PreStart()
		if err != nil {
			return err
		}

		go func() {
			for errCg := range c.cg.Errors() {
				xlog.Error(c.ctx, c.logPrefix, xlog.Err(fmt.Errorf("client error: %v", errCg)))
			}
		}()

		eg, ctx := errgroup.WithContext(c.ctx)

		eg.Go(func() error {
			for {
				if err := c.cg.Consume(ctx, []string{c.topic}, c.handler); err != nil {
					xlog.Warn(c.ctx, c.logPrefix, xlog.Err(fmt.Errorf("error start consumer: %v", err)))
				}
				if err := c.ctx.Err(); err != nil {
					return fmt.Errorf("context was canceled: %w", err)
				}
			}
		})

		return eg.Wait()
	}
}

func (c *BaseConsumer) Stop() graceful.ProcessStopper {
	return func(ctx context.Context) error {
		if err := c.cg.Close(); err != nil {
			return err
		}
		return nil
	}
}
