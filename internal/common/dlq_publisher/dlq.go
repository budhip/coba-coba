package dlqpublisher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/Shopify/sarama"
)

const prefixLogMessage = "[DLQ]"

type Publisher interface {
	Publish(message models.FailedMessage) error
}

type kafkaDlq struct {
	producer sarama.SyncProducer
	topic    string
	metrics  metrics.Metrics
}

func New(p sarama.SyncProducer, topic string, metrics metrics.Metrics) Publisher {
	return kafkaDlq{p, topic, metrics}
}

func (d kafkaDlq) Publish(message models.FailedMessage) (err error) {
	startTime := time.Now()
	defer func() {
		if d.metrics != nil {
			d.metrics.GetPublisherPrometheus().GenerateMetrics(startTime, d.topic, err)
		}
	}()

	msg, err := d.prepareMessage(message)
	if err != nil {
		xlog.Error(
			context.Background(),
			prefixLogMessage,
			xlog.String("status", "prepare kafkaDlq message failed"),
			xlog.Err(err))
		return err
	}

	_, _, err = d.producer.SendMessage(msg)
	if err != nil {
		xlog.Error(
			context.Background(),
			prefixLogMessage,
			xlog.String("status", "publish kafkaDlq failed"),
			xlog.Err(err))
		return err
	}

	xlog.Info(context.Background(),
		prefixLogMessage,
		xlog.String("status", "success publish kafkaDlq message"),
		xlog.Time("timestamp", message.Timestamp),
		xlog.String("topic", d.topic),
	)

	return nil
}

func (d kafkaDlq) prepareMessage(message models.FailedMessage) (*sarama.ProducerMessage, error) {
	if message.CauseError != nil && message.Error == "" {
		message.Error = message.CauseError.Error()
	}

	msgByte, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return &sarama.ProducerMessage{
		Topic: d.topic,
		Value: sarama.ByteEncoder(msgByte),
	}, nil
}
