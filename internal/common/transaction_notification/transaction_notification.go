package transaction_notification

import (
	"context"
	"encoding/json"
	"time"

	xlog "bitbucket.org/Amartha/go-x/log"
	"bitbucket.org/Amartha/go-x/log/ctxdata"

	"github.com/Shopify/sarama"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
)

const prefixTransactionNotificationPublisherLogMessage = "[TRANSACTION-NOTIFICATION-PUBLISHER]"

type TransactionNotificationPublisher interface {
	// Publish is a method to publish transaction notification to kafka including balance logs
	Publish(ctx context.Context, payload models.TransactionNotificationPayload) error

	// PublishLogBalance is a method to publish balance logs to kafka
	PublishLogBalance(ctx context.Context, payload models.TransactionNotificationPayload) error
}

type kafkaTransactionNotification struct {
	producer           sarama.SyncProducer
	producerBalanceLog sarama.SyncProducer

	topic           string
	topicBalanceLog string
	metrics         metrics.Metrics
	cfg             config.Config
}

func NewTransactionNotificationPublisher(
	cfg config.Config,
	p sarama.SyncProducer,
	producerBalanceLogs sarama.SyncProducer,
	metrics metrics.Metrics) TransactionNotificationPublisher {

	return kafkaTransactionNotification{
		producer:           p,
		producerBalanceLog: producerBalanceLogs,
		topic:              cfg.MessageBroker.KafkaConsumer.TopicTransactionNotification,
		topicBalanceLog:    cfg.MessageBroker.KafkaConsumer.TopicBalanceLogs,
		metrics:            metrics,
		cfg:                cfg,
	}
}

func (tn kafkaTransactionNotification) Publish(ctx context.Context, payload models.TransactionNotificationPayload) (err error) {
	startTime := time.Now()
	defer func() {
		if tn.metrics != nil {
			tn.metrics.GetPublisherPrometheus().GenerateMetrics(startTime, tn.topic, err)
		}
	}()

	msg, err := tn.prepareMessage(payload)
	if err != nil {
		xlog.Error(
			ctx,
			prefixTransactionNotificationPublisherLogMessage,
			xlog.String("status", "failed to prepare message"),
			xlog.Err(err))
		return err
	}

	errPublishLogBalance := make(chan error, 1)
	go func() {
		err := tn.PublishLogBalance(ctx, payload)
		if err != nil {
			// fail open
			xlog.Error(
				ctx,
				prefixTransactionNotificationPublisherLogMessage,
				xlog.String("status", "failed to publish balance logs"),
				xlog.String("topic", tn.topic),
				xlog.String("message", string(msg.Value.(sarama.ByteEncoder))),
				xlog.Err(err))
			errPublishLogBalance <- err
			return
		}
		errPublishLogBalance <- nil
	}()

	// log the successful publishing of the transaction notification
	logDoubleMetrics(payload)

	errPublishAcuanNotif := make(chan error, 1)
	go func() {
		_, _, err := tn.producer.SendMessage(msg)
		if err != nil {
			errPublishAcuanNotif <- err
			return
		}
		errPublishAcuanNotif <- nil
	}()

	select {
	case err := <-errPublishAcuanNotif:
		// got Sarama result before timeout
		return err
	case err := <-errPublishLogBalance:
		// got Sarama result before timeout
		return err
	case <-ctx.Done():
		// client timed out first → fail open
		go func() {
			// drain late result to avoid goroutine leak, catch and log error if any
			err := <-errPublishLogBalance
			if err != nil {
				xlog.Error(
					ctx,
					prefixTransactionNotificationPublisherLogMessage,
					xlog.String("status", "failed to publish transaction notification"),
					xlog.String("topic", tn.topic),
					xlog.String("message", string(msg.Value.(sarama.ByteEncoder))),
					xlog.Err(err))
			}

			err = <-errPublishAcuanNotif
			if err != nil {
				xlog.Error(
					ctx,
					prefixTransactionNotificationPublisherLogMessage,
					xlog.String("status", "failed to publish transaction notification"),
					xlog.String("topic", tn.topic),
					xlog.String("message", string(msg.Value.(sarama.ByteEncoder))),
					xlog.Err(err))
			}
		}()

		return nil // pretend success
	}
}

func logDoubleMetrics(payload models.TransactionNotificationPayload) {
	// check if WalletTransaction nil
	// In the transaction service, the publishNotificationSuccess function does not set payload.WalletTransaction, which causes a panic.
	if payload.WalletTransaction != nil {
		walletTransactionAmount, _ := payload.WalletTransaction.NetAmount.ValueDecimal.Float64()
		xlog.LogFtm(walletTransactionAmount, "acuan-publish-transaction-notif:wallet-transaction", "acuan:wallet_transaction", string(payload.AcuanData.Body.Data.Order.OrderType), payload.ClientID)
	}

	for _, transaction := range payload.AcuanData.Body.Data.Order.Transactions {
		amountFloat, _ := transaction.Amount.Float64()
		xlog.LogFtm(amountFloat, "acuan-publish-transaction-notif:transaction", "acuan:transaction", string(transaction.TransactionType), payload.ClientID)
	}
}

func (tn kafkaTransactionNotification) PublishLogBalance(ctx context.Context, payload models.TransactionNotificationPayload) (err error) {
	payloadKafka := make(map[string][]byte)

	for accountNumber, _ := range payload.AccountBalances {
		if accountNumber == tn.cfg.AccountConfig.SystemAccountNumber {
			continue
		}

		balance, ok := payload.AccountBalances[accountNumber]
		if !ok {
			continue
		}

		payloadBytes, errMarshal := json.Marshal(models.BalanceLogsPayload{
			Before: balance.Before,
			After:  balance.After,
		})
		if errMarshal != nil {
			return errMarshal
		}

		payloadKafka[accountNumber] = payloadBytes
	}

	headers := []sarama.RecordHeader{
		{
			Key:   []byte("traceparent"),
			Value: []byte(ctxdata.GetTraceParent(ctx)),
		},
	}

	for accountNumber, payloadBytes := range payloadKafka {
		msg := &sarama.ProducerMessage{
			Headers: headers,
			Topic:   tn.topicBalanceLog,
			Key:     sarama.StringEncoder(accountNumber),
			Value:   sarama.ByteEncoder(payloadBytes),
		}

		_, _, err = tn.producerBalanceLog.SendMessage(msg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (tn kafkaTransactionNotification) prepareMessage(payload models.TransactionNotificationPayload) (*sarama.ProducerMessage, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	msg := &sarama.ProducerMessage{
		Topic: tn.topic,
		Value: sarama.ByteEncoder(payloadBytes),
	}

	return msg, nil
}
