package publisher

import (
	"hash"
	"time"

	"github.com/Shopify/sarama"
)

type Option func(*sarama.Config)

func NewKafkaSyncProducer(brokers []string, opts ...Option) (sarama.SyncProducer, error) {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.Return.Errors = true
	saramaCfg.Producer.Timeout = 2 * time.Second
	saramaCfg.Net.DialTimeout = 2 * time.Second
	saramaCfg.Net.ReadTimeout = 2 * time.Second
	saramaCfg.Net.WriteTimeout = 2 * time.Second

	for _, opt := range opts {
		opt(saramaCfg)
	}

	producer, err := sarama.NewSyncProducer(brokers, saramaCfg)
	if err != nil {
		return nil, err
	}

	return producer, nil
}

func WithCustomHasher(hasher func() hash.Hash32) Option {
	return func(cfg *sarama.Config) {
		cfg.Producer.Partitioner = sarama.NewCustomHashPartitioner(hasher)
	}
}
