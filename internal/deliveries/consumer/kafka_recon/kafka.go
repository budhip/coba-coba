package kafkarecon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/config"

	goacuanlib "bitbucket.org/Amartha/go-acuan-lib/model"
	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/Shopify/sarama"
)

var logMessage = "[KAFKA-CONSUMER-RECON] "

type ReconConsumer struct {
	ctx           context.Context
	ctxCancel     context.CancelFunc
	consumerCfg   config.ConsumerConfig
	client        sarama.ConsumerGroup
	initialOffset int64

	processor func(transactions []goacuanlib.Transaction)
	dateLimit time.Time
}

func NewReconConsumer(ctx context.Context, cfg config.Config, p func(transactions []goacuanlib.Transaction), dl time.Time, i int64) (*ReconConsumer, error) {
	c := &ReconConsumer{
		ctx:           context.Background(),
		consumerCfg:   cfg.MessageBroker.KafkaConsumer,
		processor:     p,
		dateLimit:     dl,
		initialOffset: i,
	}

	if len(c.consumerCfg.Brokers) == 0 {
		xlog.Error(context.Background(), "no kafka bootstrap brokers defined, please set the brokers")
		return nil, errors.New("no kafka bootstrap brokers defined, please set the brokers")
	}

	if c.consumerCfg.Topic == "" {
		return nil, errors.New("no topics given to be consumed, please set the topic")
	}

	if c.consumerCfg.ConsumerGroup == "" {
		return nil, errors.New("no kafka consumer group defined, please set the group")
	}

	xlog.Info(c.ctx, logMessage, xlog.String("status", "success init kafka recon consumer"))

	return c, nil
}

func (c *ReconConsumer) preStart() error {
	c.ctx, c.ctxCancel = context.WithCancel(c.ctx)

	/**
	 * Construct a new Sarama configuration.
	 * The Kafka cluster version has to be defined before the consumer/producer is initialized.
	 */
	config := sarama.NewConfig()
	config.Version = sarama.V3_0_0_0
	config.ClientID = c.consumerCfg.ConsumerGroupDailyRecon

	if c.consumerCfg.IsVerbose {
		sarama.Logger = log.New(os.Stdout, logMessage, log.LstdFlags)
	}
	config.Consumer.Offsets.Initial = c.initialOffset

	switch c.consumerCfg.Assignor {
	case "sticky":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategySticky}
	case "roundrobin":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRoundRobin}
	case "range":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	default:
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	}

	client, err := sarama.NewConsumerGroup(c.consumerCfg.Brokers, c.consumerCfg.ConsumerGroupDailyRecon, config)
	if err != nil {
		return err
	}
	c.client = client

	return nil
}

func (c *ReconConsumer) Start() {
	if err := c.preStart(); err != nil {
		xlog.Error(c.ctx, logMessage, xlog.Err(fmt.Errorf("failed to init consumer group: %v", xlog.Err(err))))
		return
	}

	// track errors
	go func() {
		for err := range c.client.Errors() {
			xlog.Error(c.ctx, logMessage, xlog.Err(fmt.Errorf("client error: %v", xlog.Err(err))))
		}
	}()

	// `Consume` should be called inside an infinite loop, when a
	// server-side rebalance happens, the consumer session will need to be
	// recreated to get the new claims
	if err := c.client.Consume(c.ctx, []string{c.consumerCfg.Topic}, c); err != nil {
		xlog.Error(c.ctx, logMessage, xlog.Err(fmt.Errorf("error when consume: %v", err)))
	}
	// check if context was cancelled, signaling that the consumer should stop
	if c.ctx.Err() != nil {
		return
	}
}

func (c *ReconConsumer) Stop() {
	c.ctxCancel()
	if c.client != nil {
		if err := c.client.Close(); err != nil {
			xlog.Error(c.ctx, logMessage, xlog.Err(fmt.Errorf("error closing client: %v", err)))
		}
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *ReconConsumer) Setup(ses sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	// close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *ReconConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *ReconConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// Setup to stop claims
	var idleDuration = time.Second * 3
	idleTimer := time.NewTimer(idleDuration)
	defer idleTimer.Stop()

ConsumerLoop:
	for {
		idleTimer.Reset(idleDuration)
		select {
		case message := <-claim.Messages():
			var data goacuanlib.Payload[goacuanlib.DataOrder]
			if err := json.Unmarshal(message.Value, &data); err != nil {
				xlog.Error(c.ctx, logMessage, xlog.Err(fmt.Errorf("error unmarshal json %v, value %v", err, string(message.Value))))
				break
			}

			if data.Body.Data.Order.OrderTime.After(c.dateLimit) {
				break ConsumerLoop
			}
			c.processor(data.Body.Data.Order.Transactions)

			session.MarkMessage(message, "")

			continue

		case <-idleTimer.C:
			break ConsumerLoop
		}
	}

	return nil
}
