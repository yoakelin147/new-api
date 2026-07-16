package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCostStatisticsAggregatesConsumeLogs(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&Channel{Id: 11, Name: "primary-channel", Key: "key-11"}).Error)
	require.NoError(t, DB.Create(&Channel{Id: 22, Name: "backup-channel", Key: "key-22"}).Error)

	base := time.Date(2026, 7, 16, 9, 0, 0, 0, time.Local)
	logs := []*Log{
		{
			UserId:           7,
			CreatedAt:        base.Unix(),
			Type:             LogTypeConsume,
			TokenId:          101,
			TokenName:        "codex-key",
			ModelName:        "gpt-5.5",
			Group:            "default",
			ChannelId:        11,
			Quota:            100,
			PromptTokens:     10,
			CompletionTokens: 20,
			Other:            `{"group_ratio":0.5}`,
		},
		{
			UserId:           7,
			CreatedAt:        base.Add(time.Minute).Unix(),
			Type:             LogTypeConsume,
			TokenId:          101,
			TokenName:        "codex-key",
			ModelName:        "gpt-5.5",
			Group:            "default",
			ChannelId:        11,
			Quota:            200,
			PromptTokens:     30,
			CompletionTokens: 40,
			Other:            `{"group_ratio":0.5}`,
		},
		{
			UserId:           7,
			CreatedAt:        base.Add(2 * time.Minute).Unix(),
			Type:             LogTypeConsume,
			TokenId:          202,
			TokenName:        "backup-key",
			ModelName:        "glm-5",
			Group:            "vip",
			ChannelId:        22,
			Quota:            50,
			PromptTokens:     5,
			CompletionTokens: 6,
			Other:            `{"group_ratio":0.8}`,
		},
		{
			UserId:    8,
			CreatedAt: base.Add(3 * time.Minute).Unix(),
			Type:      LogTypeConsume,
			TokenId:   303,
			TokenName: "other-user-key",
			ModelName: "gpt-5.5",
			Group:     "default",
			ChannelId: 11,
			Quota:     999,
		},
		{
			UserId:    7,
			CreatedAt: base.Add(4 * time.Minute).Unix(),
			Type:      LogTypeError,
			TokenId:   101,
			TokenName: "codex-key",
			ModelName: "gpt-5.5",
			Group:     "default",
			ChannelId: 11,
			Quota:     999,
		},
	}
	require.NoError(t, DB.Create(&logs).Error)

	rows, total, summary, err := GetCostStatistics(CostStatisticsQuery{
		UserId:         7,
		IsAdmin:        false,
		LogType:        LogTypeConsume,
		StartTimestamp: base.Add(-time.Hour).Unix(),
		EndTimestamp:   base.Add(time.Hour).Unix(),
	}, 0, 10)
	require.NoError(t, err)

	assert.Equal(t, 2, total)
	assert.Equal(t, CostStatisticsSummary{
		Quota:            350,
		RequestCount:     3,
		PromptTokens:     45,
		CompletionTokens: 66,
		TotalTokens:      111,
	}, summary)

	byToken := make(map[string]*CostStatisticsRow)
	for _, row := range rows {
		byToken[row.TokenName] = row
	}

	assert.Equal(t, &CostStatisticsRow{
		Date:             base.Format("20060102"),
		Type:             LogTypeConsume,
		TokenId:          101,
		TokenName:        "codex-key",
		ModelName:        "gpt-5.5",
		Group:            "default",
		ChannelId:        11,
		ChannelName:      "primary-channel",
		PromptTokens:     40,
		CompletionTokens: 60,
		TotalTokens:      100,
		RequestCount:     2,
		Quota:            300,
		GroupRatio:       0.5,
	}, byToken["codex-key"])
	assert.Equal(t, "backup-channel", byToken["backup-key"].ChannelName)
	assert.Equal(t, 0.8, byToken["backup-key"].GroupRatio)
}

func TestGetCostStatisticsPagination(t *testing.T) {
	truncateTables(t)

	base := time.Date(2026, 7, 16, 9, 0, 0, 0, time.Local)
	logs := []*Log{
		{UserId: 7, CreatedAt: base.Unix(), Type: LogTypeConsume, TokenName: "a", ModelName: "m1", Quota: 100},
		{UserId: 7, CreatedAt: base.Unix(), Type: LogTypeConsume, TokenName: "b", ModelName: "m2", Quota: 200},
	}
	require.NoError(t, DB.Create(&logs).Error)

	rows, total, summary, err := GetCostStatistics(CostStatisticsQuery{
		UserId:         7,
		IsAdmin:        false,
		LogType:        LogTypeConsume,
		StartTimestamp: base.Add(-time.Hour).Unix(),
		EndTimestamp:   base.Add(time.Hour).Unix(),
	}, 0, 1)
	require.NoError(t, err)

	assert.Equal(t, 2, total)
	assert.Len(t, rows, 1)
	assert.Equal(t, 300, summary.Quota)
}
