package messaging

import (
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/messaging/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type kafkaConfigTestHelper struct {
	mockCtrl *gomock.Controller

	group string
	topic string

	broker *sarama.MockBroker
}

func (th kafkaConfigTestHelper) close() {
	th.broker.Close()
	th.mockCtrl.Finish()
}

func newKafkaTestHelper(t *testing.T) kafkaConfigTestHelper {
	t.Helper()
	t.Parallel()

	var (
		group = "go-fp-transaction"
		topic = "test"
	)

	mockCtrl := gomock.NewController(t)

	broker := mock.NewMockBroker(t, group, topic)

	return kafkaConfigTestHelper{
		mockCtrl: mockCtrl,
		group:    group,
		topic:    topic,
		broker:   broker,
	}
}

func Test_createSaramaConsumerConfig(t *testing.T) {

	th := newKafkaTestHelper(t)
	defer th.close()

	type args struct {
		cfg config.ConsumerConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success create config",
			args: args{
				cfg: config.ConsumerConfig{
					Brokers:       []string{th.broker.Addr()},
					Topic:         th.topic,
					ConsumerGroup: th.group,
					IsOldest:      true,
				},
			},
			wantErr: false,
		},
		{
			name: "success using assignor sticky",
			args: args{
				cfg: config.ConsumerConfig{
					Brokers:       []string{th.broker.Addr()},
					Topic:         th.topic,
					ConsumerGroup: th.group,
					Assignor:      "sticky",
				},
			},
			wantErr: false,
		},
		{
			name: "success using assignor roundrobin",
			args: args{
				cfg: config.ConsumerConfig{
					Brokers:       []string{th.broker.Addr()},
					Topic:         th.topic,
					ConsumerGroup: th.group,
					Assignor:      "roundrobin",
				},
			},
			wantErr: false,
		},
		{
			name: "success using assignor range",
			args: args{
				cfg: config.ConsumerConfig{
					Brokers:       []string{th.broker.Addr()},
					Topic:         th.topic,
					ConsumerGroup: th.group,
					Assignor:      "range",
				},
			},
			wantErr: false,
		},
		{
			name: "error missing broker",
			args: args{
				cfg: config.ConsumerConfig{
					Brokers:       nil,
					Topic:         th.topic,
					ConsumerGroup: th.group,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateSaramaConsumerConfig(tt.args.cfg, "[TEST]")
			assert.Equal(t, tt.wantErr, err != nil, err)
		})
	}
}
