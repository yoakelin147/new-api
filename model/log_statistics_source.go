package model

import (
	"strings"

	"gorm.io/gorm"
)

func validateLogTextPatterns(values ...string) error {
	for _, value := range values {
		if strings.Contains(value, "%") {
			if _, err := sanitizeLikePattern(value); err != nil {
				return err
			}
		}
	}
	return nil
}

func costSQLQuery(query CostStatisticsQuery) (*gorm.DB, error) {
	tx := LOG_DB.Model(&Log{}).Where("logs.created_at >= ? AND logs.created_at <= ?", query.StartTimestamp, query.EndTimestamp)
	if query.LogType == LogTypeUnknown {
		tx = tx.Where("logs.type IN ?", []int{LogTypeConsume, LogTypeRefund})
	} else {
		tx = tx.Where("logs.type = ?", query.LogType)
	}
	if !query.IsAdmin && query.UserIds != nil {
		tx = tx.Where("logs.user_id IN ?", query.UserIds)
	} else if !query.IsAdmin {
		tx = tx.Where("logs.user_id = ?", query.UserId)
	} else if query.Username != "" {
		var err error
		tx, err = applyExplicitLogTextFilter(tx, "logs.username", query.Username)
		if err != nil {
			return nil, err
		}
	}
	if query.TokenName != "" {
		tx = tx.Where("logs.token_name = ?", query.TokenName)
	}
	if query.ModelName != "" {
		var err error
		tx, err = applyExplicitLogTextFilter(tx, "logs.model_name", query.ModelName)
		if err != nil {
			return nil, err
		}
	}
	if query.IsAdmin && query.ChannelId != 0 {
		tx = tx.Where("logs.channel_id = ?", query.ChannelId)
	}
	if query.Group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", query.Group)
	}

	tx = tx.Select("id, created_at, type, username, token_id, token_name, model_name, quota, prompt_tokens, completion_tokens, channel_id, use_time, " + logGroupCol + ", other")
	return tx, nil
}

func profitSQLQuery(query ProfitStatisticsQuery) (*gorm.DB, error) {
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
			return nil, err
		}
	}
	if query.ModelName != "" {
		var err error
		tx, err = applyExplicitLogTextFilter(tx, "logs.model_name", query.ModelName)
		if err != nil {
			return nil, err
		}
	}
	if query.ChannelId != 0 {
		tx = tx.Where("logs.channel_id = ?", query.ChannelId)
	}
	if query.Group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", query.Group)
	}
	return tx, nil
}

func costLogFilter(query CostStatisticsQuery) logFilter {
	f := logFilter{Types: []int{LogTypeConsume, LogTypeRefund}, Start: query.StartTimestamp, End: query.EndTimestamp, TokenName: query.TokenName, ModelName: query.ModelName, Group: query.Group}
	if query.LogType != LogTypeUnknown {
		f.Types = []int{query.LogType}
	}
	if query.IsAdmin {
		f.Username = query.Username
		f.ChannelID = query.ChannelId
	} else if query.UserIds != nil {
		f.UserIDs = query.UserIds
	} else {
		f.UserIDs = []int{query.UserId}
	}
	return f
}

func profitLogFilter(query ProfitStatisticsQuery) logFilter {
	f := logFilter{Types: []int{LogTypeConsume, LogTypeRefund}, Start: query.StartTimestamp, End: query.EndTimestamp, Username: query.Username, ModelName: query.ModelName, ChannelID: query.ChannelId, Group: query.Group}
	if query.LogType != LogTypeUnknown {
		f.Types = []int{query.LogType}
	}
	return f
}

func forEachCostLog(query CostStatisticsQuery, visit func(Log) error) error {
	if clickHouseLogs != nil {
		return clickHouseLogs.Scan(logReadContext(query.ReadState), costLogFilter(query), visit)
	}
	tx, err := costSQLQuery(query)
	if err != nil {
		return err
	}
	return scanSQLLogs(tx, visit)
}

func forEachProfitLog(query ProfitStatisticsQuery, visit func(Log) error) error {
	if clickHouseLogs != nil {
		return clickHouseLogs.Scan(logReadContext(query.ReadState), profitLogFilter(query), visit)
	}
	tx, err := profitSQLQuery(query)
	if err != nil {
		return err
	}
	tx = tx.Select("id, created_at, type, username, model_name, quota, prompt_tokens, completion_tokens, channel_id, " + logGroupCol + ", other")
	return scanSQLLogs(tx, visit)
}

func scanSQLLogs(tx *gorm.DB, visit func(Log) error) error {
	var batch []Log
	return tx.FindInBatches(&batch, 1000, func(_ *gorm.DB, _ int) error {
		for _, log := range batch {
			if err := visit(log); err != nil {
				return err
			}
		}
		return nil
	}).Error
}

func profitLogChannelIDs(query ProfitStatisticsQuery) (map[int]struct{}, error) {
	if clickHouseLogs != nil {
		return clickHouseLogs.ChannelIDs(logReadContext(query.ReadState), profitLogFilter(query))
	}
	tx, err := profitSQLQuery(query)
	if err != nil {
		return nil, err
	}
	var ids []int
	if err := tx.Distinct("logs.channel_id").Where("logs.channel_id <> ?", 0).Pluck("logs.channel_id", &ids).Error; err != nil {
		return nil, err
	}
	result := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		result[id] = struct{}{}
	}
	return result, nil
}
