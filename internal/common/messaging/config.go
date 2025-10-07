package messaging

import (
	"context"
	"errors"
	"log"
	"os"

	xlog "bitbucket.org/Amartha/go-x/log"
	"github.com/Shopify/sarama"

	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
)

func CreateSaramaConsumerConfig(cfg config.ConsumerConfig, logPrefix string) (*sarama.Config, error) {
	if len(cfg.Brokers) == 0 {
		xlog.Error(context.Background(), "no kafka bootstrap brokers defined, please set the brokers")
		return nil, errors.New("no kafka bootstrap brokers defined, please set the brokers")
	}

	saramaCfg := sarama.NewConfig()
	saramaCfg.Version = sarama.V3_0_0_0
	saramaCfg.ClientID, _ = os.Hostname()
	saramaCfg.Consumer.Return.Errors = true

	if cfg.IsVerbose {
		sarama.Logger = log.New(os.Stdout, logPrefix, log.LstdFlags)
	}

	if cfg.IsOldest {
		saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	switch cfg.Assignor {
	case "sticky":
		saramaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategySticky}
	case "roundrobin":
		saramaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRoundRobin}
	case "range":
		saramaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	default:
		saramaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	}

	return saramaCfg, nil
}
