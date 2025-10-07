package kafkaconsumer

import (
	"context"
	"encoding/json"
	"fmt"

	xlog "bitbucket.org/Amartha/go-x/log"

	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/Shopify/sarama"
)

const prefixJournalPublisherLogMessage = "[JOURNAL-PUBLISHER]"

type JournalPublisher interface {
	Publish(ctx context.Context, payload *models.JournalStreamPayload) error
}

type kafkaJournal struct {
	producer sarama.SyncProducer
	topic    string
}

func NewJournalPublisher(cfg config.Config, p sarama.SyncProducer) JournalPublisher {
	return kafkaJournal{
		producer: p,
		topic:    cfg.MessageBroker.KafkaConsumer.TopicAccountingJournal,
	}
}

func (p kafkaJournal) Publish(ctx context.Context, payload *models.JournalStreamPayload) error {
	msg, err := p.prepareMessage(payload)
	if err != nil {
		xlog.Error(
			ctx,
			prefixJournalPublisherLogMessage,
			xlog.String("status", "failed to prepare message"),
			xlog.Err(err))
		return err
	}

	_, _, err = p.producer.SendMessage(msg)
	if err != nil {
		xlog.Error(
			ctx,
			prefixJournalPublisherLogMessage,
			xlog.String("status", "failed to send message"),
			xlog.Err(err))
		return err
	}

	xlog.Info(ctx,
		prefixJournalPublisherLogMessage,
		xlog.String("status", "success publish journal accounting"),
		xlog.String("topic", p.topic),
	)

	return nil
}

func (p kafkaJournal) prepareMessage(payload *models.JournalStreamPayload) (*sarama.ProducerMessage, error) {
	msgByte, err := json.Marshal(*payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return &sarama.ProducerMessage{Topic: p.topic, Value: sarama.ByteEncoder(msgByte)}, nil
}
