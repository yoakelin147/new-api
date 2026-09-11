package model

import (
	"context"
	"sort"
)

func clickHouseUsageFilter(start, end int64) logFilter {
	f := logFilter{Types: []int{LogTypeConsume}}
	if start > 0 {
		f.Start = start
	}
	if end > 0 {
		f.End = end
	}
	return f
}

func clickHouseChannelSeries(start, end int64) ([]*ChannelQuotaData, error) {
	filter := clickHouseUsageFilter(start, end)
	filter.NonzeroChannel = true
	type key struct {
		channel int
		hour    int64
	}
	byKey := map[key]*ChannelQuotaData{}
	ids := map[int]bool{}
	err := clickHouseLogs.Scan(context.Background(), filter, func(log Log) error {
		k := key{log.ChannelId, log.CreatedAt - log.CreatedAt%3600}
		row := byKey[k]
		if row == nil {
			row = &ChannelQuotaData{ChannelId: k.channel, CreatedAt: k.hour}
			byKey[k] = row
		}
		row.Count++
		row.Quota += log.Quota
		row.TokenUsed += log.PromptTokens + log.CompletionTokens
		ids[log.ChannelId] = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	channelIDs := make([]int, 0, len(ids))
	for id := range ids {
		channelIDs = append(channelIDs, id)
	}
	names, err := getChannelNamesByIds(channelIDs)
	if err != nil {
		return nil, err
	}
	rows := make([]*ChannelQuotaData, 0, len(byKey))
	for _, row := range byKey {
		row.ChannelName = names[row.ChannelId]
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt != rows[j].CreatedAt {
			return rows[i].CreatedAt < rows[j].CreatedAt
		}
		return rows[i].ChannelId < rows[j].ChannelId
	})
	return rows, nil
}

type tokenSeriesKind int

const (
	tokenSeriesSelf tokenSeriesKind = iota
	tokenSeriesUsers
)

func clickHouseTokenSeries(userID int, start, end int64, kind tokenSeriesKind) ([]*UserTokenQuotaData, error) {
	filter := clickHouseUsageFilter(start, end)
	filter.HasToken = true
	if kind == tokenSeriesSelf || userID > 0 {
		filter.UserIDs = []int{userID}
	} else {
		filter.NonzeroUser = true
	}
	type key struct {
		userID, tokenID     int
		username, tokenName string
		hour                int64
	}
	byKey := map[key]*UserTokenQuotaData{}
	err := clickHouseLogs.Scan(context.Background(), filter, func(log Log) error {
		username := log.Username
		if kind == tokenSeriesSelf {
			username = ""
		}
		k := key{log.UserId, log.TokenId, username, log.TokenName, log.CreatedAt - log.CreatedAt%3600}
		row := byKey[k]
		if row == nil {
			row = &UserTokenQuotaData{UserID: k.userID, Username: k.username, TokenId: k.tokenID, TokenName: k.tokenName, CreatedAt: k.hour}
			byKey[k] = row
		}
		row.Count++
		row.Quota += log.Quota
		row.TokenUsed += log.PromptTokens + log.CompletionTokens
		return nil
	})
	if err != nil {
		return nil, err
	}
	rows := make([]*UserTokenQuotaData, 0, len(byKey))
	for _, row := range byKey {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt != rows[j].CreatedAt {
			return rows[i].CreatedAt < rows[j].CreatedAt
		}
		if rows[i].UserID != rows[j].UserID {
			return rows[i].UserID < rows[j].UserID
		}
		if rows[i].TokenId != rows[j].TokenId {
			return rows[i].TokenId < rows[j].TokenId
		}
		return rows[i].TokenName < rows[j].TokenName
	})
	return rows, nil
}
