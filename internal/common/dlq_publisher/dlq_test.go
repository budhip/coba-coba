package dlqpublisher

import (
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mockSarama "github.com/Shopify/sarama/mocks"
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

func TestNewDLQ(t *testing.T) {
	th := newConsumerTestHelper(t)

	type args struct {
		p     sarama.SyncProducer
		topic string
	}

	tests := []struct {
		name string
		args args
		want Publisher
	}{
		{
			name: "success new DLQ",
			args: args{
				p:     th.producer,
				topic: th.topic,
			},
			want: kafkaDlq{
				producer: th.producer,
				topic:    th.topic,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, New(tt.args.p, tt.args.topic, nil), "New(%v, %v)", tt.args.p, tt.args.topic)
		})
	}
}

func Test_dlq_Publish(t *testing.T) {
	th := newConsumerTestHelper(t)
	defer func() {
		th.broker.Close()
		th.mockCtrl.Finish()
	}()

	type fields struct {
		producer sarama.SyncProducer
		topic    string
	}
	type args struct {
		message models.FailedMessage
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		doMock  func(a args)
		wantErr bool
	}{
		{
			name: "success publish message",
			fields: fields{
				producer: th.producer,
				topic:    th.topic,
			},
			args: args{
				message: models.FailedMessage{
					Timestamp:  time.Now(),
					Payload:    []byte(`{"key": "value"}`),
					CauseError: assert.AnError,
				},
			},
			doMock: func(a args) {
				th.producer.ExpectSendMessageAndSucceed()
			},
			wantErr: false,
		},
		{
			name: "success publish message without giving error",
			fields: fields{
				producer: th.producer,
				topic:    th.topic,
			},
			args: args{
				message: models.FailedMessage{
					Timestamp:  time.Now(),
					Payload:    []byte(`{"key": "value"}`),
					CauseError: nil,
				},
			},
			doMock: func(a args) {
				th.producer.ExpectSendMessageAndSucceed()
			},
			wantErr: false,
		},
		{
			name: "failed publish message",
			fields: fields{
				producer: th.producer,
				topic:    th.topic,
			},
			args: args{
				message: models.FailedMessage{
					Timestamp:  time.Now(),
					Payload:    []byte(`{"key": "value"}`),
					CauseError: nil,
				},
			},
			doMock: func(a args) {
				th.producer.ExpectSendMessageAndFail(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			d := kafkaDlq{
				producer: tt.fields.producer,
				topic:    tt.fields.topic,
			}
			err := d.Publish(tt.args.message)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
