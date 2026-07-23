package model

import (
	"errors"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

type CostStatisticsQuery struct {
	UserId         int
	IsAdmin        bool
	LogType        int
	StartTimestamp int64
	EndTimestamp   int64
	Username       string
	TokenName      string
	ModelName      string
	ChannelId      int
	Group          string
}

type CostStatisticsSummary struct {
	Quota            int `json:"quota"`
	RequestCount     int `json:"request_count"`
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type CostStatisticsRow struct {
	Date             string  `json:"date"`
	Type             int     `json:"type"`
	TokenId          int     `json:"token_id"`
	TokenName        string  `json:"token_name"`
	ModelName        string  `json:"model_name"`
	Group            string  `json:"group"`
	Username         string  `json:"username"`
	ChannelId        int     `json:"channel_id"`
	ChannelName      string  `json:"channel_name"`
	PromptTokens     int     `json:"prompt_tokens"`
	InputTokens      int     `json:"input_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	RequestCount     int     `json:"request_count"`
	Quota            int     `json:"quota"`
	GroupRatio       float64 `json:"group_ratio"`
	ModelRatio       float64 `json:"model_ratio"`
	CompletionRatio  float64 `json:"completion_ratio"`
	CacheTokens      int     `json:"cache_tokens"`
	CacheRatio       float64 `json:"cache_ratio"`
	CacheWrite5m     int     `json:"cache_write_5m_tokens"`
	CacheWrite5mRate float64 `json:"cache_write_5m_ratio"`
	CacheWrite1h     int     `json:"cache_write_1h_tokens"`
	CacheWrite1hRate float64 `json:"cache_write_1h_ratio"`
	ModelPrice       float64 `json:"model_price"`
	BillingMode      string  `json:"billing_mode"`
	MatchedTier      string  `json:"matched_tier"`
	UseTimeSeconds   int     `json:"use_time_seconds"`
}

type costStatisticsKey struct {
	Date        string
	Type        int
	TokenId     int
	TokenName   string
	ModelName   string
	Group       string
	Username    string
	ChannelId   int
	ChannelName string
}

type costStatisticsLogOther struct {
	ModelRatio            float64 `json:"model_ratio"`
	GroupRatio            float64 `json:"group_ratio"`
	CompletionRatio       float64 `json:"completion_ratio"`
	CacheTokens           int     `json:"cache_tokens"`
	CacheRatio            float64 `json:"cache_ratio"`
	CacheCreationTokens   int     `json:"cache_creation_tokens"`
	CacheCreationRatio    float64 `json:"cache_creation_ratio"`
	CacheCreationTokens5m int     `json:"cache_creation_tokens_5m"`
	CacheCreationRatio5m  float64 `json:"cache_creation_ratio_5m"`
	CacheCreationTokens1h int     `json:"cache_creation_tokens_1h"`
	CacheCreationRatio1h  float64 `json:"cache_creation_ratio_1h"`
	ModelPrice            float64 `json:"model_price"`
	BillingMode           string  `json:"billing_mode"`
	MatchedTier           string  `json:"matched_tier"`
	UsageSemantic         string  `json:"usage_semantic"`
	Claude                bool    `json:"claude"`
}

func GetCostStatistics(query CostStatisticsQuery, startIdx int, pageSize int) ([]*CostStatisticsRow, int, CostStatisticsSummary, error) {
	rowsByKey := make(map[costStatisticsKey]*CostStatisticsRow)
	channelIds := make(map[int]struct{})
	summary := CostStatisticsSummary{}

	tx := LOG_DB.Model(&Log{}).Where("logs.created_at >= ? AND logs.created_at <= ?", query.StartTimestamp, query.EndTimestamp)
	if query.LogType == LogTypeUnknown {
		tx = tx.Where("logs.type IN ?", []int{LogTypeConsume, LogTypeRefund})
	} else {
		tx = tx.Where("logs.type = ?", query.LogType)
	}
	if !query.IsAdmin {
		tx = tx.Where("logs.user_id = ?", query.UserId)
	} else if query.Username != "" {
		var err error
		tx, err = applyExplicitLogTextFilter(tx, "logs.username", query.Username)
		if err != nil {
			return nil, 0, summary, err
		}
	}
	if query.TokenName != "" {
		tx = tx.Where("logs.token_name = ?", query.TokenName)
	}
	if query.ModelName != "" {
		var err error
		tx, err = applyExplicitLogTextFilter(tx, "logs.model_name", query.ModelName)
		if err != nil {
			return nil, 0, summary, err
		}
	}
	if query.IsAdmin && query.ChannelId != 0 {
		tx = tx.Where("logs.channel_id = ?", query.ChannelId)
	}
	if query.Group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", query.Group)
	}

	tx = tx.Select("id, created_at, type, username, token_id, token_name, model_name, quota, prompt_tokens, completion_tokens, channel_id, use_time, " + logGroupCol + ", other")

	var batch []Log
	err := tx.FindInBatches(&batch, 1000, func(_tx *gorm.DB, _batch int) error {
		for _, log := range batch {
			date := time.Unix(log.CreatedAt, 0).Local().Format("20060102")
			channelId := 0
			if query.IsAdmin {
				channelId = log.ChannelId
			}
			key := costStatisticsKey{
				Date:      date,
				Type:      log.Type,
				TokenId:   log.TokenId,
				TokenName: log.TokenName,
				ModelName: log.ModelName,
				Group:     log.Group,
				Username:  log.Username,
				ChannelId: channelId,
			}
			row := rowsByKey[key]
			if row == nil {
				row = &CostStatisticsRow{
					Date:       date,
					Type:       log.Type,
					TokenId:    log.TokenId,
					TokenName:  log.TokenName,
					ModelName:  log.ModelName,
					Group:      log.Group,
					Username:   log.Username,
					ChannelId:  channelId,
					GroupRatio: 0,
				}
				rowsByKey[key] = row
			}
			other := parseCostStatisticsLogOther(log.Other)
			quota := costStatisticsSignedQuota(log)
			row.PromptTokens += log.PromptTokens
			row.InputTokens += costStatisticsInputTokens(log.PromptTokens, other)
			row.CompletionTokens += log.CompletionTokens
			row.TotalTokens += log.PromptTokens + log.CompletionTokens
			row.RequestCount++
			row.Quota += quota
			row.UseTimeSeconds += log.UseTime
			row.CacheTokens += other.CacheTokens
			row.CacheWrite5m += costStatisticsCacheWrite5mTokens(other)
			row.CacheWrite1h += other.CacheCreationTokens1h
			applyCostStatisticsOther(row, other)

			summary.PromptTokens += log.PromptTokens
			summary.CompletionTokens += log.CompletionTokens
			summary.TotalTokens += log.PromptTokens + log.CompletionTokens
			summary.RequestCount++
			summary.Quota += quota
			if query.IsAdmin && log.ChannelId != 0 {
				channelIds[log.ChannelId] = struct{}{}
			}
		}
		return nil
	}).Error
	if err != nil {
		common.SysError("failed to query cost statistics: " + err.Error())
		return nil, 0, summary, errors.New("查询费用统计失败")
	}

	channelNames := map[int]string{}
	if query.IsAdmin {
		var err error
		channelNames, err = getChannelNameMap(channelIds)
		if err != nil {
			return nil, 0, summary, err
		}
	}

	rows := make([]*CostStatisticsRow, 0, len(rowsByKey))
	for _, row := range rowsByKey {
		if query.IsAdmin {
			row.ChannelName = channelNames[row.ChannelId]
		}
		if row.RequestCount > 0 {
			row.UseTimeSeconds = row.UseTimeSeconds / row.RequestCount
		}
		rows = append(rows, row)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Date != rows[j].Date {
			return rows[i].Date > rows[j].Date
		}
		if rows[i].Quota != rows[j].Quota {
			return rows[i].Quota > rows[j].Quota
		}
		if rows[i].TokenName != rows[j].TokenName {
			return rows[i].TokenName < rows[j].TokenName
		}
		if rows[i].ModelName != rows[j].ModelName {
			return rows[i].ModelName < rows[j].ModelName
		}
		return rows[i].ChannelId < rows[j].ChannelId
	})

	total := len(rows)
	if startIdx >= total {
		return []*CostStatisticsRow{}, total, summary, nil
	}
	endIdx := startIdx + pageSize
	if endIdx > total {
		endIdx = total
	}
	return rows[startIdx:endIdx], total, summary, nil
}

func costStatisticsSignedQuota(log Log) int {
	if log.Type == LogTypeRefund {
		return -log.Quota
	}
	return log.Quota
}

func parseCostStatisticsLogOther(other string) costStatisticsLogOther {
	if other == "" {
		return costStatisticsLogOther{}
	}
	var data costStatisticsLogOther
	if err := common.UnmarshalJsonStr(other, &data); err != nil {
		return costStatisticsLogOther{}
	}
	return data
}

func costStatisticsInputTokens(promptTokens int, other costStatisticsLogOther) int {
	if other.UsageSemantic == "anthropic" || other.Claude {
		return promptTokens
	}
	inputTokens := promptTokens - other.CacheTokens - other.CacheCreationTokens - other.CacheCreationTokens5m - other.CacheCreationTokens1h
	if inputTokens < 0 {
		return 0
	}
	return inputTokens
}

func costStatisticsCacheWrite5mTokens(other costStatisticsLogOther) int {
	if other.CacheCreationTokens5m > 0 {
		return other.CacheCreationTokens5m
	}
	return other.CacheCreationTokens
}

func applyCostStatisticsOther(row *CostStatisticsRow, other costStatisticsLogOther) {
	if row.GroupRatio == 0 && other.GroupRatio != 0 {
		row.GroupRatio = other.GroupRatio
	}
	if row.ModelRatio == 0 && other.ModelRatio != 0 {
		row.ModelRatio = other.ModelRatio
	}
	if row.CompletionRatio == 0 && other.CompletionRatio != 0 {
		row.CompletionRatio = other.CompletionRatio
	}
	if row.CacheRatio == 0 && other.CacheRatio != 0 {
		row.CacheRatio = other.CacheRatio
	}
	if row.CacheWrite5mRate == 0 {
		if other.CacheCreationRatio5m != 0 {
			row.CacheWrite5mRate = other.CacheCreationRatio5m
		} else if other.CacheCreationRatio != 0 {
			row.CacheWrite5mRate = other.CacheCreationRatio
		}
	}
	if row.CacheWrite1hRate == 0 && other.CacheCreationRatio1h != 0 {
		row.CacheWrite1hRate = other.CacheCreationRatio1h
	}
	if row.ModelPrice == 0 && other.ModelPrice > 0 {
		row.ModelPrice = other.ModelPrice
	}
	if other.BillingMode == "tiered_expr" || row.BillingMode == "" {
		row.BillingMode = costStatisticsBillingMode(other)
	}
	if row.MatchedTier == "" && other.MatchedTier != "" {
		row.MatchedTier = other.MatchedTier
	}
}

func costStatisticsBillingMode(other costStatisticsLogOther) string {
	switch other.BillingMode {
	case "tiered_expr":
		return "动态计费"
	}
	if other.ModelPrice > 0 {
		return "按次计费"
	}
	return "按量倍率"
}

func getChannelNameMap(channelIds map[int]struct{}) (map[int]string, error) {
	names := make(map[int]string, len(channelIds))
	if len(channelIds) == 0 {
		return names, nil
	}
	ids := make([]int, 0, len(channelIds))
	for id := range channelIds {
		ids = append(ids, id)
	}
	var channels []struct {
		Id   int
		Name string
	}
	if err := DB.Table("channels").Select("id, name").Where("id IN ?", ids).Find(&channels).Error; err != nil {
		common.SysError("failed to query cost statistics channel names: " + err.Error())
		return nil, errors.New("查询渠道名称失败")
	}
	for _, channel := range channels {
		names[channel.Id] = channel.Name
	}
	return names, nil
}
