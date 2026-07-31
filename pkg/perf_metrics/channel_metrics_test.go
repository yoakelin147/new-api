package perfmetrics

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestBuildChannelSummaryResultRanksByHiddenRequestCount(t *testing.T) {
	totals := map[channelModelKey]counters{
		{model: "model-b", channelId: 2}: {
			requestCount:   3,
			successCount:   2,
			totalLatencyMs: 900,
			outputTokens:   30,
			generationMs:   1000,
		},
		{model: "model-a", channelId: 1}: {
			requestCount:   5,
			successCount:   5,
			totalLatencyMs: 1000,
			outputTokens:   50,
			generationMs:   2000,
		},
		{model: "model-a", channelId: 3}: {
			requestCount:   1,
			successCount:   0,
			totalLatencyMs: 500,
		},
	}

	result := buildChannelSummaryResult(totals, map[int]string{
		1: "channel-1",
		2: "channel-2",
		3: "channel-3",
	}, 2)

	require.Equal(t, []ChannelModelSummary{
		{
			ModelName:    "model-a",
			ChannelId:    1,
			ChannelName:  "channel-1",
			AvgLatencyMs: 200,
			SuccessRate:  100,
			AvgTps:       25,
			RequestCount: 5,
		},
		{
			ModelName:    "model-b",
			ChannelId:    2,
			ChannelName:  "channel-2",
			AvgLatencyMs: 300,
			SuccessRate:  66.67,
			AvgTps:       30,
			RequestCount: 3,
		},
	}, result.Models)
}

func TestChannelSummaryKeepsSameModelOnDifferentChannels(t *testing.T) {
	result := buildChannelSummaryResult(map[channelModelKey]counters{
		{model: "shared-model", channelId: 1}: {requestCount: 1},
		{model: "shared-model", channelId: 2}: {requestCount: 1},
	}, map[int]string{1: "first", 2: "second"}, 20)

	require.Len(t, result.Models, 2)
	require.Equal(t, 1, result.Models[0].ChannelId)
	require.Equal(t, 2, result.Models[1].ChannelId)
}

func TestChannelSummaryJSONOmitsInternalRequestCount(t *testing.T) {
	result := buildChannelSummaryResult(map[channelModelKey]counters{
		{model: "model", channelId: 1}: {requestCount: 2, successCount: 2},
	}, map[int]string{1: "channel"}, 20)

	encoded, err := common.Marshal(result)

	require.NoError(t, err)
	require.JSONEq(t, `{
		"models": [{
			"model_name": "model",
			"channel_id": 1,
			"channel_name": "channel",
			"avg_latency_ms": 0,
			"success_rate": 100,
			"avg_tps": 0
		}]
	}`, string(encoded))
}

func TestChannelBucketStartUsesFixedFiveMinuteBuckets(t *testing.T) {
	require.Equal(t, int64(0), channelBucketStart(299))
	require.Equal(t, int64(300), channelBucketStart(300))
	require.Equal(t, int64(300), channelBucketStart(599))
}
