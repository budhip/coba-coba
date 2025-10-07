package setup

import (
	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/transaction_notification"
)

type PublisherClient struct {
	TransactionNotification transaction_notification.TransactionNotificationPublisher
	TransactionDQL          dlqpublisher.Publisher
}
