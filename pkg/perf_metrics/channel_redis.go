package perfmetrics

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/perf_metrics_setting"
	"github.com/go-redis/redis/v8"
)

const (
	channelBucketSeconds     int64 = 300
	channelMinuteSeconds     int64 = 60
	channelRedisTTL                = 25 * time.Hour
	channelRedisWriteTimeout       = time.Second
	channelRedisReadTimeout        = 3 * time.Second
	channelRedisErrorLogGap        = time.Minute
)

var (
	errChannelMetricsRedisUnavailable = errors.New("channel performance metrics require Redis")
	lastChannelRedisWriteError        atomic.Int64
)

func RecordChannelFailure(modelName string, channelId int, latencyMs int64) {
	RecordChannel(Sample{
		Model:     modelName,
		ChannelId: channelId,
		LatencyMs: latencyMs,
	})
}

// RecordChannel records model-channel metrics in Redis. Callers invoke this
// from an asynchronous metrics path so Redis latency never blocks relaying.
func RecordChannel(sample Sample) {
	setting := perf_metrics_setting.GetSetting()
	if !setting.Enabled || sample.Model == "" || sample.ChannelId <= 0 {
		return
	}
	if !channelMetricsRedisAvailable() {
		logChannelRedisWriteError(errChannelMetricsRedisUnavailable)
		return
	}
	if sample.LatencyMs < 0 {
		sample.LatencyMs = 0
	}

	now := time.Now().Unix()
	bucketTs := channelBucketStart(now)
	minuteOffset := channelMinuteOffset(now)
	ctx, cancel := context.WithTimeout(context.Background(), channelRedisWriteTimeout)
	defer cancel()

	redisKey := channelRedisBucketKey(bucketTs)
	fieldPrefix := channelRedisFieldPrefix(sample.Model, sample.ChannelId, minuteOffset)
	pipe := common.RDB.TxPipeline()
	pipe.HIncrBy(ctx, redisKey, fieldPrefix+":req", 1)
	if sample.Success {
		pipe.HIncrBy(ctx, redisKey, fieldPrefix+":ok", 1)
	}
	if sample.LatencyMs > 0 {
		pipe.HIncrBy(ctx, redisKey, fieldPrefix+":lat", sample.LatencyMs)
	}
	if sample.OutputTokens > 0 && sample.GenerationMs > 0 {
		pipe.HIncrBy(ctx, redisKey, fieldPrefix+":out", sample.OutputTokens)
		pipe.HIncrBy(ctx, redisKey, fieldPrefix+":gen_ms", sample.GenerationMs)
	}
	pipe.Expire(ctx, redisKey, channelRedisTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		logChannelRedisWriteError(err)
	}
}

func QueryChannelSummary(minutes int, limit int) (ChannelSummaryResult, error) {
	if minutes <= 0 {
		minutes = 5
	}
	if minutes > 60*24 {
		minutes = 60 * 24
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if !channelMetricsRedisAvailable() {
		return ChannelSummaryResult{}, errChannelMetricsRedisUnavailable
	}

	totals, err := queryChannelRedisTotals(time.Now().Unix(), minutes)
	if err != nil {
		return ChannelSummaryResult{}, err
	}

	result := buildChannelSummaryResult(totals, nil, limit)
	channelIds := make([]int, 0, len(result.Models))
	seenChannelIds := make(map[int]struct{}, len(result.Models))
	for _, item := range result.Models {
		if _, ok := seenChannelIds[item.ChannelId]; ok {
			continue
		}
		seenChannelIds[item.ChannelId] = struct{}{}
		channelIds = append(channelIds, item.ChannelId)
	}
	channelNames, err := model.GetChannelNamesByIds(channelIds)
	if err != nil {
		return ChannelSummaryResult{}, err
	}
	for index := range result.Models {
		result.Models[index].ChannelName = channelNames[result.Models[index].ChannelId]
	}

	return result, nil
}

func queryChannelRedisTotals(now int64, minutes int) (map[channelModelKey]counters, error) {
	if !channelMetricsRedisAvailable() {
		return nil, errChannelMetricsRedisUnavailable
	}
	firstMinute, lastMinute := channelMinuteWindow(now, minutes)
	bucketStarts := channelRedisBucketStarts(firstMinute, lastMinute)
	ctx, cancel := context.WithTimeout(context.Background(), channelRedisReadTimeout)
	defer cancel()

	pipe := common.RDB.Pipeline()
	commands := make([]*redis.StringStringMapCmd, 0, len(bucketStarts))
	for _, bucketTs := range bucketStarts {
		commands = append(commands, pipe.HGetAll(ctx, channelRedisBucketKey(bucketTs)))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("query channel performance metrics from Redis: %w", err)
	}

	totals := make(map[channelModelKey]counters)
	for index, command := range commands {
		mergeChannelRedisBucket(totals, bucketStarts[index], firstMinute, lastMinute, command.Val())
	}
	return totals, nil
}

func buildChannelSummaryResult(totals map[channelModelKey]counters, channelNames map[int]string, limit int) ChannelSummaryResult {
	models := make([]ChannelModelSummary, 0, len(totals))
	for key, total := range totals {
		if total.requestCount == 0 {
			continue
		}
		models = append(models, ChannelModelSummary{
			ModelName:    key.model,
			ChannelId:    key.channelId,
			ChannelName:  channelNames[key.channelId],
			AvgLatencyMs: avg(total.totalLatencyMs, total.requestCount),
			SuccessRate:  math.Round(successRate(total)*100) / 100,
			AvgTps:       math.Round(avgTps(total)*100) / 100,
			RequestCount: total.requestCount,
		})
	}

	sort.Slice(models, func(i, j int) bool {
		if models[i].RequestCount != models[j].RequestCount {
			return models[i].RequestCount > models[j].RequestCount
		}
		if models[i].ModelName != models[j].ModelName {
			return models[i].ModelName < models[j].ModelName
		}
		return models[i].ChannelId < models[j].ChannelId
	})
	if limit > 0 && len(models) > limit {
		models = models[:limit]
	}

	return ChannelSummaryResult{Models: models}
}

func mergeChannelRedisBucket(totals map[channelModelKey]counters, bucketTs int64, firstMinute int64, lastMinute int64, values map[string]string) {
	for field, rawValue := range values {
		key, minuteOffset, metric, ok := parseChannelRedisField(field)
		if !ok {
			continue
		}
		minuteTs := bucketTs + int64(minuteOffset)*channelMinuteSeconds
		if minuteTs < firstMinute || minuteTs > lastMinute {
			continue
		}
		value, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil || value < 0 {
			continue
		}
		current := totals[key]
		switch metric {
		case "req":
			current.requestCount += value
		case "ok":
			current.successCount += value
		case "lat":
			current.totalLatencyMs += value
		case "out":
			current.outputTokens += value
		case "gen_ms":
			current.generationMs += value
		default:
			continue
		}
		totals[key] = current
	}
}

func channelMinuteWindow(now int64, minutes int) (int64, int64) {
	lastMinute := now - now%channelMinuteSeconds
	firstMinute := lastMinute - int64(minutes-1)*channelMinuteSeconds
	return firstMinute, lastMinute
}

func channelRedisBucketStarts(firstMinute int64, lastMinute int64) []int64 {
	firstBucket := channelBucketStart(firstMinute)
	lastBucket := channelBucketStart(lastMinute)
	buckets := make([]int64, 0, (lastBucket-firstBucket)/channelBucketSeconds+1)
	for bucketTs := firstBucket; bucketTs <= lastBucket; bucketTs += channelBucketSeconds {
		buckets = append(buckets, bucketTs)
	}
	return buckets
}

func channelBucketStart(ts int64) int64 {
	return ts - ts%channelBucketSeconds
}

func channelMinuteOffset(ts int64) int {
	return int((ts - channelBucketStart(ts)) / channelMinuteSeconds)
}

func channelRedisBucketKey(bucketTs int64) string {
	return fmt.Sprintf("perf:channel:v1:%d", bucketTs)
}

func channelRedisFieldPrefix(modelName string, channelId int, minuteOffset int) string {
	encodedModel := base64.RawURLEncoding.EncodeToString([]byte(modelName))
	return fmt.Sprintf("%s:%d:%d", encodedModel, channelId, minuteOffset)
}

func parseChannelRedisField(field string) (channelModelKey, int, string, bool) {
	parts := strings.Split(field, ":")
	if len(parts) != 4 {
		return channelModelKey{}, 0, "", false
	}
	decodedModel, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(decodedModel) == 0 {
		return channelModelKey{}, 0, "", false
	}
	channelId, err := strconv.Atoi(parts[1])
	if err != nil || channelId <= 0 {
		return channelModelKey{}, 0, "", false
	}
	minuteOffset, err := strconv.Atoi(parts[2])
	if err != nil || minuteOffset < 0 || minuteOffset >= int(channelBucketSeconds/channelMinuteSeconds) {
		return channelModelKey{}, 0, "", false
	}
	return channelModelKey{model: string(decodedModel), channelId: channelId}, minuteOffset, parts[3], true
}

func channelMetricsRedisAvailable() bool {
	return common.RedisEnabled && common.RDB != nil
}

func logChannelRedisWriteError(err error) {
	now := time.Now().UnixNano()
	minimumGap := channelRedisErrorLogGap.Nanoseconds()
	previous := lastChannelRedisWriteError.Load()
	if now-previous < minimumGap || !lastChannelRedisWriteError.CompareAndSwap(previous, now) {
		return
	}
	common.SysError("failed to record channel performance metrics in Redis: " + err.Error())
}
