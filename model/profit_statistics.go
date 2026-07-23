package model

import (
	"errors"
	"math"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	ProfitCostStatusConfigured   = "configured"
	ProfitCostStatusUnconfigured = "unconfigured"
	ProfitCostStatusEstimated    = "estimated"
)

type ProfitStatisticsQuery struct {
	LogType        int
	StartTimestamp int64
	EndTimestamp   int64
	Username       string
	ModelName      string
	ChannelId      int
	Group          string
}

type ProfitStatisticsSummary struct {
	RequestCount       int     `json:"request_count"`
	PromptTokens       int     `json:"prompt_tokens"`
	CompletionTokens   int     `json:"completion_tokens"`
	TotalTokens        int     `json:"total_tokens"`
	CustomerAmount     float64 `json:"customer_amount"`
	OriginalAmount     float64 `json:"original_amount"`
	UpstreamCost       float64 `json:"upstream_cost"`
	Profit             float64 `json:"profit"`
	ProfitRate         float64 `json:"profit_rate"`
	UnconfiguredRows   int     `json:"unconfigured_rows"`
	UnconfiguredAmount float64 `json:"unconfigured_amount"`
	EstimatedRows      int     `json:"estimated_rows"`
}

type ProfitStatisticsRow struct {
	Date              string   `json:"date"`
	Username          string   `json:"username"`
	Group             string   `json:"group"`
	ChannelId         int      `json:"channel_id"`
	ChannelName       string   `json:"channel_name"`
	ModelName         string   `json:"model_name"`
	RequestCount      int      `json:"request_count"`
	PromptTokens      int      `json:"prompt_tokens"`
	CompletionTokens  int      `json:"completion_tokens"`
	TotalTokens       int      `json:"total_tokens"`
	CustomerRatio     float64  `json:"customer_ratio"`
	UpstreamRatio     *float64 `json:"upstream_ratio"`
	CustomerAmount    float64  `json:"customer_amount"`
	OriginalAmount    float64  `json:"original_amount"`
	UpstreamCost      *float64 `json:"upstream_cost"`
	Profit            *float64 `json:"profit"`
	ProfitRate        *float64 `json:"profit_rate"`
	CostStatus        string   `json:"cost_status"`
	MissingRatioCount int      `json:"missing_ratio_count"`
}

type profitStatisticsKey struct {
	Date             string
	Username         string
	Group            string
	ChannelId        int
	ModelName        string
	CustomerRatioKey int64
}

type profitStatisticsAccumulator struct {
	row              *ProfitStatisticsRow
	upstreamCost     float64
	hasUpstreamRatio bool
}

func GetProfitStatistics(query ProfitStatisticsQuery, startIdx int, pageSize int) ([]*ProfitStatisticsRow, int, ProfitStatisticsSummary, error) {
	rowsByKey := make(map[profitStatisticsKey]*profitStatisticsAccumulator)
	channelIds := make(map[int]struct{})
	summary := ProfitStatisticsSummary{}

	tx := LOG_DB.Model(&Log{}).Where("logs.created_at >= ? AND logs.created_at <= ?", query.StartTimestamp, query.EndTimestamp)
	if query.LogType == LogTypeUnknown {
		tx = tx.Where("logs.type IN ?", []int{LogTypeConsume, LogTypeRefund})
	} else {
		tx = tx.Where("logs.type = ?", query.LogType)
	}
	if query.Username != "" {
		var err error
		tx, err = applyExplicitLogTextFilter(tx, "logs.username", query.Username)
		if err != nil {
			return nil, 0, summary, err
		}
	}
	if query.ModelName != "" {
		var err error
		tx, err = applyExplicitLogTextFilter(tx, "logs.model_name", query.ModelName)
		if err != nil {
			return nil, 0, summary, err
		}
	}
	if query.ChannelId != 0 {
		tx = tx.Where("logs.channel_id = ?", query.ChannelId)
	}
	if query.Group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", query.Group)
	}

	var channelIdList []int
	if err := tx.Session(&gorm.Session{}).Distinct("logs.channel_id").Where("logs.channel_id <> ?", 0).Pluck("logs.channel_id", &channelIdList).Error; err != nil {
		common.SysError("failed to query profit statistics channel ids: " + err.Error())
		return nil, 0, summary, errors.New("查询渠道利润统计失败")
	}
	for _, channelId := range channelIdList {
		channelIds[channelId] = struct{}{}
	}

	ratiosByChannel, err := getEnabledChannelProfitRatios(channelIds)
	if err != nil {
		return nil, 0, summary, err
	}
	channelNames, err := getChannelNameMap(channelIds)
	if err != nil {
		return nil, 0, summary, err
	}

	tx = tx.Select("id, created_at, type, username, model_name, quota, prompt_tokens, completion_tokens, channel_id, " + logGroupCol + ", other")
	var logs []Log
	if err := tx.FindInBatches(&logs, 1000, func(_tx *gorm.DB, _batch int) error {
		for _, log := range logs {
			addProfitStatisticsLog(rowsByKey, ratiosByChannel, channelNames, log)
		}
		return nil
	}).Error; err != nil {
		common.SysError("failed to aggregate profit statistics logs: " + err.Error())
		return nil, 0, summary, errors.New("查询渠道利润统计失败")
	}

	rows := make([]*ProfitStatisticsRow, 0, len(rowsByKey))
	for _, acc := range rowsByKey {
		row := acc.row
		if row.OriginalAmount != 0 {
			row.CustomerRatio = row.CustomerAmount / row.OriginalAmount
		} else {
			row.CustomerRatio = 1
		}
		if acc.hasUpstreamRatio {
			upstreamCost := acc.upstreamCost
			profit := row.CustomerAmount - upstreamCost
			row.UpstreamCost = &upstreamCost
			row.Profit = &profit
			if row.CustomerAmount != 0 {
				profitRate := profit / row.CustomerAmount
				row.ProfitRate = &profitRate
			}
			row.CostStatus = ProfitCostStatusConfigured
			if row.MissingRatioCount > 0 {
				row.CostStatus = ProfitCostStatusEstimated
			}
			summary.UpstreamCost += upstreamCost
			summary.Profit += profit
		} else {
			summary.UnconfiguredRows++
			summary.UnconfiguredAmount += row.CustomerAmount
		}
		if row.MissingRatioCount > 0 {
			summary.EstimatedRows++
		}
		summary.RequestCount += row.RequestCount
		summary.PromptTokens += row.PromptTokens
		summary.CompletionTokens += row.CompletionTokens
		summary.TotalTokens += row.TotalTokens
		summary.CustomerAmount += row.CustomerAmount
		summary.OriginalAmount += row.OriginalAmount
		rows = append(rows, row)
	}
	configuredCustomerAmount := summary.CustomerAmount - summary.UnconfiguredAmount
	if configuredCustomerAmount != 0 {
		summary.ProfitRate = summary.Profit / configuredCustomerAmount
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Date != rows[j].Date {
			return rows[i].Date > rows[j].Date
		}
		if rows[i].Profit != nil && rows[j].Profit != nil && *rows[i].Profit != *rows[j].Profit {
			return *rows[i].Profit > *rows[j].Profit
		}
		if rows[i].CustomerAmount != rows[j].CustomerAmount {
			return rows[i].CustomerAmount > rows[j].CustomerAmount
		}
		if rows[i].ChannelId != rows[j].ChannelId {
			return rows[i].ChannelId < rows[j].ChannelId
		}
		if rows[i].ModelName != rows[j].ModelName {
			return rows[i].ModelName < rows[j].ModelName
		}
		return rows[i].Username < rows[j].Username
	})

	total := len(rows)
	if startIdx >= total {
		return []*ProfitStatisticsRow{}, total, summary, nil
	}
	endIdx := startIdx + pageSize
	if endIdx > total {
		endIdx = total
	}
	return rows[startIdx:endIdx], total, summary, nil
}

func addProfitStatisticsLog(rowsByKey map[profitStatisticsKey]*profitStatisticsAccumulator, ratiosByChannel map[int][]ChannelProfitRatio, channelNames map[int]string, log Log) {
	date := time.Unix(log.CreatedAt, 0).Local().Format("20060102")
	username := log.Username
	other := parseCostStatisticsLogOther(log.Other)
	sign := 1.0
	if log.Type == LogTypeRefund {
		sign = -1
	}
	customerAmount := sign * float64(log.Quota) / common.QuotaPerUnit
	customerRatio := other.GroupRatio
	missingRatio := false
	if customerRatio <= 0 {
		customerRatio = 1
		missingRatio = true
	}
	originalAmount := customerAmount / customerRatio
	key := profitStatisticsKey{
		Date:             date,
		Username:         username,
		Group:            log.Group,
		ChannelId:        log.ChannelId,
		ModelName:        log.ModelName,
		CustomerRatioKey: normalizeProfitRatioKey(customerRatio),
	}
	acc := rowsByKey[key]
	if acc == nil {
		acc = &profitStatisticsAccumulator{
			row: &ProfitStatisticsRow{
				Date:          date,
				Username:      username,
				Group:         log.Group,
				ChannelId:     log.ChannelId,
				ChannelName:   channelNames[log.ChannelId],
				ModelName:     log.ModelName,
				CustomerRatio: customerRatio,
				CostStatus:    ProfitCostStatusUnconfigured,
			},
		}
		rowsByKey[key] = acc
	}
	if missingRatio {
		acc.row.MissingRatioCount++
	}

	acc.row.RequestCount++
	acc.row.PromptTokens += log.PromptTokens
	acc.row.CompletionTokens += log.CompletionTokens
	acc.row.TotalTokens += log.PromptTokens + log.CompletionTokens
	acc.row.CustomerAmount += customerAmount
	acc.row.OriginalAmount += originalAmount

	if ratio, ok := matchChannelProfitRatio(ratiosByChannel[log.ChannelId], log.ModelName, log.Group); ok {
		upstreamCost := originalAmount * ratio.UpstreamRatio
		acc.row.UpstreamRatio = &ratio.UpstreamRatio
		acc.upstreamCost += upstreamCost
		acc.hasUpstreamRatio = true
	}
}

func normalizeProfitRatioKey(ratio float64) int64 {
	return int64(math.Round(ratio * 1_000_000_000))
}

func ProfitStatisticsHasConfiguredCost(row *ProfitStatisticsRow) bool {
	return row != nil && row.UpstreamCost != nil && row.Profit != nil
}
