package model

import (
	"context"
	"time"
)

func filterLogType(logType int) []int {
	if logType == LogTypeUnknown {
		return nil
	}
	return []int{logType}
}

func getClickHouseAdminLogs(filter logFilter, offset, limit int) ([]*Log, int64, error) {
	logs, total, err := clickHouseLogs.Query(context.Background(), filter, offset, limit, logOrderCreated)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int, 0, len(logs))
	for _, log := range logs {
		if log.ChannelId != 0 {
			ids = append(ids, log.ChannelId)
		}
	}
	names, err := getChannelNamesByIds(ids)
	if err != nil {
		return nil, 0, err
	}
	for _, log := range logs {
		log.ChannelName = names[log.ChannelId]
	}
	return logs, total, nil
}

func sumClickHouseQuota(filter logFilter) (stat Stat, err error) {
	filter.Types = []int{LogTypeConsume}
	err = clickHouseLogs.Scan(context.Background(), filter, func(log Log) error {
		stat.Quota += log.Quota
		return nil
	})
	if err != nil {
		return stat, err
	}
	filter.Start = time.Now().Add(-60 * time.Second).Unix()
	filter.End = 0
	err = clickHouseLogs.Scan(context.Background(), filter, func(log Log) error {
		stat.Rpm++
		stat.Tpm += log.PromptTokens + log.CompletionTokens
		return nil
	})
	return stat, err
}
