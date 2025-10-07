package publisher

import (
	"context"
	"encoding/json"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/Shopify/sarama"
)

const logIdentifier = "[GENERAL-PUBLISHER]"

type Publisher interface {
	Publish(ctx context.Context, message any, opts ...PublishOption) error
}

type publishOptions struct {
	key     string
	headers map[string]string
}

type PublishOption func(*publishOptions)

func WithKey(key string) PublishOption {
	return func(opts *publishOptions) {
		opts.key = key
	}
}

func WithHeaders(headers map[string]string) PublishOption {
	return func(opts *publishOptions) {
		opts.headers = headers
	}
}

type publisher struct {
	producer sarama.SyncProducer
	topic    string
}

func NewPublisher(p sarama.SyncProducer, topic string) Publisher {
	return publisher{
		producer: p,
		topic:    topic,
	}
}

func (d publisher) Publish(ctx context.Context, message any, opts ...PublishOption) error {
	options := &publishOptions{}
	for _, opt := range opts {
		opt(options)
	}

	msg, err := d.prepareMessage(message, options)
	if err != nil {
		xlog.Error(
			ctx,
			logIdentifier,
			xlog.String("status", "failed prepare message"),
			xlog.Err(err))
		return err
	}

	_, _, err = d.producer.SendMessage(msg)
	if err != nil {
		xlog.Error(
			ctx,
			logIdentifier,
			xlog.String("status", "failed send message"),
			xlog.Err(err))
		return err
	}

	xlog.Info(ctx,
		logIdentifier,
		xlog.String("status", "success publish message"),
		xlog.Time("timestamp", common.Now()),
		xlog.String("topic", d.topic),
	)

	return nil
}

func (d publisher) prepareMessage(message any, opts *publishOptions) (*sarama.ProducerMessage, error) {
	msgByte, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	producerMsg := &sarama.ProducerMessage{
		Topic: d.topic,
		Value: sarama.ByteEncoder(msgByte),
	}

	if opts != nil {
		if opts.key != "" {
			producerMsg.Key = sarama.StringEncoder(opts.key)
		}

		if len(opts.headers) > 0 {
			var headers []sarama.RecordHeader
			for key, value := range opts.headers {
				headers = append(headers, sarama.RecordHeader{
					Key:   []byte(key),
					Value: []byte(value),
				})
			}

			producerMsg.Headers = headers
		}
	}

	return producerMsg, nil
}
