package kafkaconsumer

import (
	"context"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/messaging/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/Shopify/sarama"
	mockSarama "github.com/Shopify/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type consumerTestHelper struct {
	mockCtrl *gomock.Controller
	broker   *sarama.MockBroker
	producer *mockSarama.SyncProducer
	topic    string
}

func newConsumerTestHelper(t *testing.T) consumerTestHelper {
	var (
		topic = "test.consumer.topic"
		group = "test.consumer.group"
	)

	mockCtrl := gomock.NewController(t)

	broker := mock.NewMockBroker(t, group, topic)
	sp := mockSarama.NewSyncProducer(t, nil)

	return consumerTestHelper{
		mockCtrl: mockCtrl,
		broker:   broker,
		topic:    topic,
		producer: sp,
	}
}

func TestNewJournalPublisher(t *testing.T) {
	th := newConsumerTestHelper(t)

	tests := []struct {
		name string
		want JournalPublisher
	}{
		{
			name: "happy path",
			want: kafkaJournal{
				producer: th.producer,
				topic:    th.topic,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{
				MessageBroker: config.MessageBroker{
					KafkaConsumer: config.ConsumerConfig{
						TopicAccountingJournal: th.topic,
					},
				},
			}
			assert.Equalf(t, tc.want, NewJournalPublisher(cfg, th.producer), "NewJournalPublisher(%v, %v)", th.producer, th.topic)
		})
	}
}

func TestJournalPublish(t *testing.T) {
	th := newConsumerTestHelper(t)

	defer func() {
		th.broker.Close()
		th.mockCtrl.Finish()
	}()

	tests := []struct {
		name    string
		doMock  func()
		wantErr bool
	}{
		{
			name: "success publish message",
			doMock: func() {
				th.producer.ExpectSendMessageAndSucceed()
			},
			wantErr: false,
		},
		{
			name: "failed publish message",
			doMock: func() {
				th.producer.ExpectSendMessageAndFail(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			d := kafkaJournal{
				producer: th.producer,
				topic:    th.topic,
			}
			err := d.Publish(context.Background(), &models.JournalStreamPayload{})
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}
