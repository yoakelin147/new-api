package model

import (
	"errors"
	"fmt"
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
	ChannelId        int     `json:"channel_id"`
	ChannelName      string  `json:"channel_name"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	RequestCount     int     `json:"request_count"`
	Quota            int     `json:"quota"`
	GroupRatio       float64 `json:"group_ratio"`
	UseTimeSeconds   int     `json:"use_time_seconds"`
}

type costStatisticsKey struct {
	Date        string
	Type        int
	TokenId     int
	TokenName   string
	ModelName   string
	Group       string
	ChannelId   int
	ChannelName string
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

	tx = tx.Select("id, created_at, type, token_id, token_name, model_name, quota, prompt_tokens, completion_tokens, channel_id, use_time, " + logGroupCol + ", other")

	var batch []Log
	err := tx.Order("logs.created_at desc, logs.id desc").FindInBatches(&batch, 1000, func(_tx *gorm.DB, _batch int) error {
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
					ChannelId:  channelId,
					GroupRatio: 0,
				}
				rowsByKey[key] = row
			}
			row.PromptTokens += log.PromptTokens
			row.CompletionTokens += log.CompletionTokens
			row.TotalTokens += log.PromptTokens + log.CompletionTokens
			row.RequestCount++
			row.Quota += log.Quota
			row.UseTimeSeconds += log.UseTime
			if row.GroupRatio == 0 {
				if ratio, ok := parseLogGroupRatio(log.Other); ok {
					row.GroupRatio = ratio
				}
			}

			summary.PromptTokens += log.PromptTokens
			summary.CompletionTokens += log.CompletionTokens
			summary.TotalTokens += log.PromptTokens + log.CompletionTokens
			summary.RequestCount++
			summary.Quota += log.Quota
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

func parseLogGroupRatio(other string) (float64, bool) {
	if other == "" {
		return 0, false
	}
	var data map[string]interface{}
	if err := common.UnmarshalJsonStr(other, &data); err != nil {
		return 0, false
	}
	value, ok := data["group_ratio"]
	if !ok {
		return 0, false
	}
	switch ratio := value.(type) {
	case float64:
		return ratio, true
	case int:
		return float64(ratio), true
	case string:
		var parsed float64
		if _, err := fmt.Sscanf(ratio, "%f", &parsed); err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
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
