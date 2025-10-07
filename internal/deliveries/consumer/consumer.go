package consumer

import (
	"context"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/cmd/setup"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/graceful"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/publisher"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/account_mutation"
	"bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/dlq_notification"
	hvtbalanceupdate "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/hvt_balance_update"
	"bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/process_wallet_transaction"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	dlqretrier "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/dlq_retrier"
	kafkaconsumer "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/kafka"
	queuerecon "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/task_queue_recon"
)

func NewKafkaConsumer(
	ctx context.Context,
	consumerName string,
	conf config.Config,
	svc *services.Services,
	cacheRepo repositories.CacheRepository,
	contract *setup.Setup,
) (consumerProcess graceful.ProcessStartStopper, stoppers []graceful.ProcessStopper, err error) {
	switch consumerName {
	case "transaction":
		producer, errProducer := publisher.NewKafkaSyncProducer(conf.MessageBroker.KafkaConsumer.Brokers)
		if errProducer != nil {
			err = errProducer
			return
		}

		stoppers = append(stoppers, func(ctx context.Context) error { return producer.Close() })

		trxJournal := kafkaconsumer.NewJournalPublisher(conf, producer) // TODO: Unused on injected service

		consumerProcess, err = kafkaconsumer.New(
			ctx,
			conf,
			svc.Transaction,
			contract.PublisherClient.TransactionDQL,
			trxJournal,
			svc.Account,
			contract.PublisherClient.TransactionNotification,
			contract.Metrics,
		)
	case "dlq_notification":
		consumerProcess, err = dlq_notification.New(ctx, conf, svc.DLQProcessor, contract.Metrics)
	case "dlq_retrier":
		consumerProcess, err = dlqretrier.New(ctx, conf, svc.DLQProcessor, contract.Metrics)
	case "account_mutation":
		producer, errProducer := publisher.NewKafkaSyncProducer(conf.MessageBroker.KafkaConsumer.Brokers)
		if errProducer != nil {
			err = fmt.Errorf("failed setup kafka dlq publisher : %w", errProducer)
			return
		}

		stoppers = append(stoppers, func(ctx context.Context) error { return producer.Close() })

		accountDlq := dlqpublisher.New(producer, conf.MessageBroker.KafkaConsumer.TopicAccountMutationDLQ, contract.Metrics) // TODO: init publisher on setup.go
		consumerProcess, err = account_mutation.New(ctx, conf, svc.Account, accountDlq, contract.Metrics)
	case "recon_task_queue":
		consumerProcess, err = queuerecon.New(ctx, conf, services.NewReconBalanceService(svc), contract.Metrics)
	case "hvt_balance_update":
		producer, errProducer := publisher.NewKafkaSyncProducer(conf.MessageBroker.KafkaConsumer.Brokers)
		if errProducer != nil {
			err = fmt.Errorf("failed setup kafka dlq publisher : %w", errProducer)
			return
		}

		stoppers = append(stoppers, func(ctx context.Context) error { return producer.Close() })

		hvtBalanceDlq := dlqpublisher.New(producer, conf.MessageBroker.KafkaConsumer.TopicBalanceHvtDLQ, contract.Metrics)

		consumerProcess, err = hvtbalanceupdate.New(ctx, conf, svc.Balance, contract.Metrics, cacheRepo, hvtBalanceDlq)
	case "process_wallet_transaction":
		producer, errProducer := publisher.NewKafkaSyncProducer(conf.MessageBroker.KafkaConsumer.Brokers)
		if errProducer != nil {
			err = fmt.Errorf("failed setup kafka dlq publisher : %w", errProducer)
			return
		}
		processWalletDlq := dlqpublisher.New(producer, conf.MessageBroker.KafkaConsumer.TopicProcessWalletTransactionDLQ, contract.Metrics)

		consumerProcess, err = process_wallet_transaction.New(ctx, conf, contract.Metrics, cacheRepo, svc.WalletTrx, processWalletDlq)
	default:
		err = fmt.Errorf("consumer type name for %s not found", consumerName)
	}

	return
}
