package mock

import (
	"testing"

	"github.com/Shopify/sarama"
)

func NewMockBroker(t *testing.T, group, topic string) *sarama.MockBroker {
	broker := sarama.NewMockBroker(t, 0)

	// use specific handler for consumer group
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(broker.Addr(), broker.BrokerID()).
			SetLeader(topic, 0, broker.BrokerID()).
			SetController(broker.BrokerID()),
		"ApiVersionsRequest": sarama.NewMockApiVersionsResponse(t),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetOffset(topic, 0, sarama.OffsetOldest, 0).
			SetOffset(topic, 0, sarama.OffsetNewest, 1),
		"FindCoordinatorRequest": sarama.NewMockFindCoordinatorResponse(t).
			SetCoordinator(sarama.CoordinatorGroup, group, broker),
		"HeartbeatRequest": sarama.NewMockHeartbeatResponse(t),
		"JoinGroupRequest": sarama.NewMockSequence(
			sarama.NewMockJoinGroupResponse(t).SetError(sarama.ErrOffsetsLoadInProgress),
			sarama.NewMockJoinGroupResponse(t).SetGroupProtocol(sarama.RangeBalanceStrategyName),
		),
		"SyncGroupRequest": sarama.NewMockSequence(
			sarama.NewMockSyncGroupResponse(t).SetError(sarama.ErrOffsetsLoadInProgress),
			sarama.NewMockSyncGroupResponse(t).SetMemberAssignment(
				&sarama.ConsumerGroupMemberAssignment{
					Version: 0,
					Topics: map[string][]int32{
						topic: {0},
					},
				}),
		),
		"OffsetFetchRequest": sarama.NewMockOffsetFetchResponse(t).SetOffset(
			group, topic, 0, 0, "", sarama.ErrNoError,
		).SetError(sarama.ErrNoError),
		"FetchRequest": sarama.NewMockSequence(
			sarama.NewMockFetchResponse(t, 1).
				SetMessage(topic, 0, 0, sarama.StringEncoder("foo")).
				SetMessage(topic, 0, 1, sarama.StringEncoder("bar")),
			sarama.NewMockFetchResponse(t, 1),
		),
	})

	return broker
}
