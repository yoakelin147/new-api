package model

import "strings"

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
	return forEachLogs(query.ReadState, costLogFilter(query), visit)
}

func forEachProfitLog(query ProfitStatisticsQuery, visit func(Log) error) error {
	return forEachLogs(query.ReadState, profitLogFilter(query), visit)
}

func profitLogChannelIDs(query ProfitStatisticsQuery) (map[int]struct{}, error) {
	return listLogChannelIDs(query.ReadState, profitLogFilter(query))
}
