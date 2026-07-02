package model

import (
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

type TokenCacheHitStatsQuery struct {
	StartTimestamp  int64
	EndTimestamp    int64
	ModelName       string
	ChannelId       int
	TimeGranularity string
	TrendGroup      string
}

type TokenCacheHitStat struct {
	ModelName        string  `json:"model_name"`
	ChannelId        int     `json:"channel_id"`
	ChannelName      string  `json:"channel_name"`
	RequestCount     int64   `json:"request_count"`
	HitRequestCount  int64   `json:"hit_request_count"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	CacheTokens      int64   `json:"cache_tokens"`
	HitRate          float64 `json:"hit_rate"`
}

type TokenCacheHitTrendStat struct {
	ModelName        string  `json:"model_name"`
	ChannelId        int     `json:"channel_id"`
	ChannelName      string  `json:"channel_name"`
	CreatedAt        int64   `json:"created_at"`
	RequestCount     int64   `json:"request_count"`
	HitRequestCount  int64   `json:"hit_request_count"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	CacheTokens      int64   `json:"cache_tokens"`
	HitRate          float64 `json:"hit_rate"`
}

type tokenCacheLogRow struct {
	ModelName        string `gorm:"column:model_name"`
	ChannelId        int    `gorm:"column:channel_id"`
	CreatedAt        int64  `gorm:"column:created_at"`
	PromptTokens     int64  `gorm:"column:prompt_tokens"`
	CompletionTokens int64  `gorm:"column:completion_tokens"`
	Other            string `gorm:"column:other"`
}

type tokenCacheLogOther struct {
	CacheTokens int64 `json:"cache_tokens"`
}

func GetTokenCacheHitStats(params TokenCacheHitStatsQuery) ([]*TokenCacheHitStat, error) {
	rows, err := queryTokenCacheLogRows(params)
	if err != nil {
		return nil, err
	}

	statsByKey := make(map[string]*TokenCacheHitStat)
	channelIds := types.NewSet[int]()

	for _, row := range rows {
		key := row.ModelName + "\x00" + strconv.Itoa(row.ChannelId)
		stat, ok := statsByKey[key]
		if !ok {
			stat = &TokenCacheHitStat{
				ModelName: row.ModelName,
				ChannelId: row.ChannelId,
			}
			statsByKey[key] = stat
			channelIds.Add(row.ChannelId)
		}

		cacheTokens := extractTokenCacheHitTokens(row.Other)
		stat.RequestCount++
		if cacheTokens > 0 {
			stat.HitRequestCount++
		}
		stat.PromptTokens += row.PromptTokens
		stat.CompletionTokens += row.CompletionTokens
		stat.CacheTokens += cacheTokens
	}

	channelNames, err := getChannelNamesByIds(channelIds.Items())
	if err != nil {
		return nil, err
	}

	stats := make([]*TokenCacheHitStat, 0, len(statsByKey))
	for _, stat := range statsByKey {
		stat.ChannelName = channelNames[stat.ChannelId]
		if stat.PromptTokens > 0 {
			stat.HitRate = float64(stat.CacheTokens) / float64(stat.PromptTokens)
		}
		stats = append(stats, stat)
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].HitRate != stats[j].HitRate {
			return stats[i].HitRate > stats[j].HitRate
		}
		if stats[i].CacheTokens != stats[j].CacheTokens {
			return stats[i].CacheTokens > stats[j].CacheTokens
		}
		if stats[i].ChannelId != stats[j].ChannelId {
			return stats[i].ChannelId < stats[j].ChannelId
		}
		return stats[i].ModelName < stats[j].ModelName
	})

	return stats, nil
}

func GetTokenCacheHitTrendStats(params TokenCacheHitStatsQuery) ([]*TokenCacheHitTrendStat, error) {
	rows, err := queryTokenCacheLogRows(params)
	if err != nil {
		return nil, err
	}

	statsByKey := make(map[string]*TokenCacheHitTrendStat)
	channelIds := types.NewSet[int]()
	granularity := normalizeTokenCacheGranularity(params.TimeGranularity)
	trendGroup := normalizeTokenCacheTrendGroup(params.TrendGroup)

	for _, row := range rows {
		bucket := tokenCacheBucketTimestamp(row.CreatedAt, granularity)
		key := tokenCacheTrendKey(row, bucket, trendGroup)
		stat, ok := statsByKey[key]
		if !ok {
			stat = &TokenCacheHitTrendStat{
				CreatedAt: bucket,
			}
			if trendGroup == "model" {
				stat.ModelName = row.ModelName
			} else {
				stat.ChannelId = row.ChannelId
				channelIds.Add(row.ChannelId)
			}
			statsByKey[key] = stat
		}

		cacheTokens := extractTokenCacheHitTokens(row.Other)
		stat.RequestCount++
		if cacheTokens > 0 {
			stat.HitRequestCount++
		}
		stat.PromptTokens += row.PromptTokens
		stat.CompletionTokens += row.CompletionTokens
		stat.CacheTokens += cacheTokens
	}

	channelNames, err := getChannelNamesByIds(channelIds.Items())
	if err != nil {
		return nil, err
	}

	stats := make([]*TokenCacheHitTrendStat, 0, len(statsByKey))
	for _, stat := range statsByKey {
		stat.ChannelName = channelNames[stat.ChannelId]
		if stat.PromptTokens > 0 {
			stat.HitRate = float64(stat.CacheTokens) / float64(stat.PromptTokens)
		}
		stats = append(stats, stat)
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].CreatedAt != stats[j].CreatedAt {
			return stats[i].CreatedAt < stats[j].CreatedAt
		}
		if stats[i].ModelName != stats[j].ModelName {
			return stats[i].ModelName < stats[j].ModelName
		}
		return stats[i].ChannelId < stats[j].ChannelId
	})

	return stats, nil
}

func queryTokenCacheLogRows(params TokenCacheHitStatsQuery) ([]tokenCacheLogRow, error) {
	query := LOG_DB.Table("logs").
		Select("model_name, channel_id, created_at, prompt_tokens, completion_tokens, other").
		Where("type = ? and channel_id <> 0", LogTypeConsume)

	if params.ModelName != "" {
		query = query.Where("model_name LIKE ? ESCAPE '!'", fuzzyTokenCacheModelPattern(params.ModelName))
	}
	if params.StartTimestamp > 0 {
		query = query.Where("created_at >= ?", params.StartTimestamp)
	}
	if params.EndTimestamp > 0 {
		query = query.Where("created_at <= ?", params.EndTimestamp)
	}
	if params.ChannelId > 0 {
		query = query.Where("channel_id = ?", params.ChannelId)
	}

	var rows []tokenCacheLogRow
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func normalizeTokenCacheGranularity(granularity string) string {
	switch granularity {
	case "day", "week":
		return granularity
	default:
		return "hour"
	}
}

func normalizeTokenCacheTrendGroup(group string) string {
	if group == "model" {
		return "model"
	}
	return "channel"
}

func tokenCacheTrendKey(row tokenCacheLogRow, bucket int64, trendGroup string) string {
	if trendGroup == "model" {
		return row.ModelName + "\x00" + strconv.FormatInt(bucket, 10)
	}
	return strconv.Itoa(row.ChannelId) + "\x00" + strconv.FormatInt(bucket, 10)
}

func tokenCacheBucketTimestamp(timestamp int64, granularity string) int64 {
	switch granularity {
	case "day":
		return timestamp - timestamp%86400
	case "week":
		return timestamp - timestamp%604800
	default:
		return timestamp - timestamp%3600
	}
}

func fuzzyTokenCacheModelPattern(input string) string {
	input = strings.ReplaceAll(input, "!", "!!")
	input = strings.ReplaceAll(input, "%", "!%")
	input = strings.ReplaceAll(input, `_`, `!_`)
	return "%" + input + "%"
}

func extractTokenCacheHitTokens(other string) int64 {
	if other == "" {
		return 0
	}
	var parsed tokenCacheLogOther
	if err := common.Unmarshal([]byte(other), &parsed); err != nil {
		return 0
	}
	return parsed.CacheTokens
}

func getChannelNamesByIds(channelIds []int) (map[int]string, error) {
	channelNames := make(map[int]string, len(channelIds))
	if len(channelIds) == 0 {
		return channelNames, nil
	}

	missingChannelIds := channelIds
	if common.MemoryCacheEnabled {
		missingChannelIds = make([]int, 0)
		for _, channelId := range channelIds {
			cacheChannel, err := CacheGetChannel(channelId)
			if err == nil && cacheChannel != nil {
				channelNames[channelId] = cacheChannel.Name
				continue
			}
			missingChannelIds = append(missingChannelIds, channelId)
		}
	}
	if len(missingChannelIds) > 0 {
		var channels []struct {
			Id   int    `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if err := DB.Table("channels").Select("id, name").Where("id IN ?", missingChannelIds).Find(&channels).Error; err != nil {
			return channelNames, err
		}
		for _, channel := range channels {
			channelNames[channel.Id] = channel.Name
		}
	}

	return channelNames, nil
}
