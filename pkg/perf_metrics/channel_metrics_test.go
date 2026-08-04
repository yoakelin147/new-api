package perfmetrics

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestShouldRecordChannelFailure(t *testing.T) {
	tests := []struct {
		name            string
		statusCode      int
		errorCode       types.ErrorCode
		requestCanceled bool
		want            bool
	}{
		{name: "bad request", statusCode: 400, errorCode: types.ErrorCodeBadResponseStatusCode},
		{name: "unprocessable entity", statusCode: 422, errorCode: types.ErrorCodeBadResponseStatusCode},
		{name: "unauthorized upstream", statusCode: 401, errorCode: types.ErrorCodeBadResponseStatusCode, want: true},
		{name: "forbidden upstream", statusCode: 403, errorCode: types.ErrorCodeBadResponseStatusCode, want: true},
		{name: "model not found", statusCode: 404, errorCode: types.ErrorCodeModelNotFound, want: true},
		{name: "upstream timeout", statusCode: 408, errorCode: types.ErrorCodeChannelUpstreamTimeout, want: true},
		{name: "rate limited", statusCode: 429, errorCode: types.ErrorCodeBadResponseStatusCode},
		{name: "upstream server error", statusCode: 500, errorCode: types.ErrorCodeBadResponseStatusCode, want: true},
		{name: "network error", statusCode: 500, errorCode: types.ErrorCodeDoRequestFailed, want: true},
		{name: "request conversion error", statusCode: 500, errorCode: types.ErrorCodeConvertRequestFailed},
		{name: "request body error", statusCode: 500, errorCode: types.ErrorCodeBadRequestBody},
		{name: "client disconnected", statusCode: 500, errorCode: types.ErrorCodeDoRequestFailed, requestCanceled: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, ShouldRecordChannelFailure(test.statusCode, test.errorCode, test.requestCanceled))
		})
	}
}

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

func TestChannelMinuteWindowKeepsFiveRollingMinuteSlotsAcrossBucketBoundary(t *testing.T) {
	firstMinute, lastMinute := channelMinuteWindow(12*60+42, 5)

	require.Equal(t, int64(8*60), firstMinute)
	require.Equal(t, int64(12*60), lastMinute)
	require.Equal(t, []int64{5 * 60, 10 * 60}, channelRedisBucketStarts(firstMinute, lastMinute))
}

func TestChannelMinuteWindowAtFiveMinuteBoundaryIncludesPreviousFourMinutes(t *testing.T) {
	firstMinute, lastMinute := channelMinuteWindow(10*60, 5)

	require.Equal(t, int64(6*60), firstMinute)
	require.Equal(t, int64(10*60), lastMinute)
	require.Equal(t, []int64{5 * 60, 10 * 60}, channelRedisBucketStarts(firstMinute, lastMinute))
}

func TestChannelRedisFieldRoundTripSupportsArbitraryModelNames(t *testing.T) {
	field := channelRedisFieldPrefix("供应商:model/name", 42, 3) + ":gen_ms"

	key, minuteOffset, metric, ok := parseChannelRedisField(field)

	require.True(t, ok)
	require.Equal(t, channelModelKey{model: "供应商:model/name", channelId: 42}, key)
	require.Equal(t, 3, minuteOffset)
	require.Equal(t, "gen_ms", metric)
}

func TestMergeChannelRedisBucketExcludesMinuteSlotsOutsideWindow(t *testing.T) {
	totals := make(map[channelModelKey]counters)
	modelName := "model"
	channelId := 7
	values := map[string]string{
		channelRedisFieldPrefix(modelName, channelId, 0) + ":req": "10",
		channelRedisFieldPrefix(modelName, channelId, 1) + ":req": "2",
		channelRedisFieldPrefix(modelName, channelId, 1) + ":ok":  "1",
		channelRedisFieldPrefix(modelName, channelId, 1) + ":lat": "400",
	}

	mergeChannelRedisBucket(totals, 300, 360, 600, values)

	require.Equal(t, map[channelModelKey]counters{
		{model: modelName, channelId: channelId}: {
			requestCount:   2,
			successCount:   1,
			totalLatencyMs: 400,
		},
	}, totals)
}

func TestMergeChannelRedisBucketIgnoresMalformedFieldsAndValues(t *testing.T) {
	totals := make(map[channelModelKey]counters)
	validPrefix := channelRedisFieldPrefix("model", 7, 2)
	values := map[string]string{
		validPrefix + ":req":                            "3",
		validPrefix + ":ok":                             "2",
		validPrefix + ":lat":                            "900",
		validPrefix + ":out":                            "30",
		validPrefix + ":gen_ms":                         "1000",
		validPrefix + ":unknown":                        "123",
		validPrefix + ":invalid_value":                  "not-a-number",
		channelRedisFieldPrefix("model", 7, 3) + ":req": "-4",
		"invalid-field":                                 "5",
	}

	mergeChannelRedisBucket(totals, 300, 300, 599, values)

	require.Equal(t, map[channelModelKey]counters{
		{model: "model", channelId: 7}: {
			requestCount:   3,
			successCount:   2,
			totalLatencyMs: 900,
			outputTokens:   30,
			generationMs:   1000,
		},
	}, totals)
}

func TestQueryChannelSummaryRequiresRedis(t *testing.T) {
	previousClient := common.RDB
	previousEnabled := common.RedisEnabled
	common.RDB = nil
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.RDB = previousClient
		common.RedisEnabled = previousEnabled
	})

	result, err := QueryChannelSummary(5, 20)

	require.ErrorIs(t, err, errChannelMetricsRedisUnavailable)
	require.Empty(t, result.Models)
}

func TestChannelRedisReadWriteIntegration(t *testing.T) {
	redisURL := os.Getenv("CHANNEL_METRICS_REDIS_TEST_URL")
	if redisURL == "" {
		t.Skip("CHANNEL_METRICS_REDIS_TEST_URL is not set")
	}

	options, err := redis.ParseURL(redisURL)
	require.NoError(t, err)
	client := redis.NewClient(options)
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})
	require.NoError(t, client.Ping(context.Background()).Err())

	previousClient := common.RDB
	previousEnabled := common.RedisEnabled
	common.RDB = client
	common.RedisEnabled = true
	t.Cleanup(func() {
		common.RDB = previousClient
		common.RedisEnabled = previousEnabled
	})

	now := time.Now().Unix()
	redisKey := channelRedisBucketKey(channelBucketStart(now))
	require.NoError(t, client.Del(context.Background(), redisKey).Err())
	t.Cleanup(func() {
		require.NoError(t, client.Del(context.Background(), redisKey).Err())
	})

	modelName := "integration:model/渠道"
	channelId := 987654321
	RecordChannel(Sample{
		Model:        modelName,
		ChannelId:    channelId,
		LatencyMs:    1200,
		Success:      true,
		OutputTokens: 24,
		GenerationMs: 2000,
	})

	totals, err := queryChannelRedisTotals(now, 5)
	require.NoError(t, err)
	require.Equal(t, counters{
		requestCount:   1,
		successCount:   1,
		totalLatencyMs: 1200,
		outputTokens:   24,
		generationMs:   2000,
	}, totals[channelModelKey{model: modelName, channelId: channelId}])
	ttl, err := client.TTL(context.Background(), redisKey).Result()
	require.NoError(t, err)
	require.Greater(t, ttl, 24*time.Hour)
	require.LessOrEqual(t, ttl, channelRedisTTL)
}
