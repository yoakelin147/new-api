package model

import (
	"encoding/base64"
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
		PromptTokens:     40,
		InputTokens:      40,
		CompletionTokens: 60,
		TotalTokens:      100,
		RequestCount:     2,
		Quota:            300,
		GroupRatio:       0.5,
		BillingMode:      "按量倍率",
	}, byToken["codex-key"])
	assert.Zero(t, byToken["backup-key"].ChannelId)
	assert.Empty(t, byToken["backup-key"].ChannelName)
	assert.Equal(t, 0.8, byToken["backup-key"].GroupRatio)
}

func TestGetCostStatisticsSubtractsRefundQuota(t *testing.T) {
	truncateTables(t)

	base := time.Date(2026, 7, 16, 9, 0, 0, 0, time.Local)
	logs := []*Log{
		{
			UserId:    7,
			CreatedAt: base.Unix(),
			Type:      LogTypeConsume,
			TokenId:   101,
			TokenName: "codex-key",
			ModelName: "gpt-image-2",
			Group:     "default",
			Quota:     100,
		},
		{
			UserId:    7,
			CreatedAt: base.Add(time.Minute).Unix(),
			Type:      LogTypeConsume,
			TokenId:   101,
			TokenName: "codex-key",
			ModelName: "gpt-image-2",
			Group:     "default",
			Quota:     200,
		},
		{
			UserId:    7,
			CreatedAt: base.Add(2 * time.Minute).Unix(),
			Type:      LogTypeRefund,
			TokenId:   101,
			TokenName: "codex-key",
			ModelName: "gpt-image-2",
			Group:     "default",
			Quota:     50,
		},
	}
	require.NoError(t, DB.Create(&logs).Error)

	rows, total, summary, err := GetCostStatistics(CostStatisticsQuery{
		UserId:         7,
		IsAdmin:        false,
		LogType:        LogTypeUnknown,
		StartTimestamp: base.Add(-time.Hour).Unix(),
		EndTimestamp:   base.Add(time.Hour).Unix(),
	}, 0, 10)
	require.NoError(t, err)

	require.Equal(t, 2, total)
	assert.Equal(t, 250, summary.Quota)
	byType := make(map[int]*CostStatisticsRow)
	for _, row := range rows {
		byType[row.Type] = row
	}
	require.Contains(t, byType, LogTypeConsume)
	assert.Equal(t, 300, byType[LogTypeConsume].Quota)
	require.Contains(t, byType, LogTypeRefund)
	assert.Equal(t, -50, byType[LogTypeRefund].Quota)

	rows, total, summary, err = GetCostStatistics(CostStatisticsQuery{
		UserId:         7,
		IsAdmin:        false,
		LogType:        LogTypeRefund,
		StartTimestamp: base.Add(-time.Hour).Unix(),
		EndTimestamp:   base.Add(time.Hour).Unix(),
	}, 0, 10)
	require.NoError(t, err)

	require.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, -50, summary.Quota)
	assert.Equal(t, -50, rows[0].Quota)
}

func TestGetCostStatisticsHidesChannelsForSelfQuery(t *testing.T) {
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
		},
		{
			UserId:           7,
			CreatedAt:        base.Add(time.Minute).Unix(),
			Type:             LogTypeConsume,
			TokenId:          101,
			TokenName:        "codex-key",
			ModelName:        "gpt-5.5",
			Group:            "default",
			ChannelId:        22,
			Quota:            200,
			PromptTokens:     30,
			CompletionTokens: 40,
		},
	}
	require.NoError(t, DB.Create(&logs).Error)

	rows, total, summary, err := GetCostStatistics(CostStatisticsQuery{
		UserId:         7,
		IsAdmin:        false,
		LogType:        LogTypeConsume,
		StartTimestamp: base.Add(-time.Hour).Unix(),
		EndTimestamp:   base.Add(time.Hour).Unix(),
		ChannelId:      11,
	}, 0, 10)
	require.NoError(t, err)

	require.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, CostStatisticsSummary{
		Quota:            300,
		RequestCount:     2,
		PromptTokens:     40,
		CompletionTokens: 60,
		TotalTokens:      100,
	}, summary)
	assert.Equal(t, 0, rows[0].ChannelId)
	assert.Empty(t, rows[0].ChannelName)
	assert.Equal(t, 2, rows[0].RequestCount)
	assert.Equal(t, 300, rows[0].Quota)
}

func TestGetCostStatisticsKeepsChannelsForAdminQuery(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&Channel{Id: 11, Name: "primary-channel", Key: "key-11"}).Error)
	require.NoError(t, DB.Create(&Channel{Id: 22, Name: "backup-channel", Key: "key-22"}).Error)

	base := time.Date(2026, 7, 16, 9, 0, 0, 0, time.Local)
	logs := []*Log{
		{
			UserId:    7,
			CreatedAt: base.Unix(),
			Type:      LogTypeConsume,
			TokenId:   101,
			TokenName: "codex-key",
			ModelName: "gpt-5.5",
			Group:     "default",
			ChannelId: 11,
			Quota:     100,
		},
		{
			UserId:    7,
			CreatedAt: base.Add(time.Minute).Unix(),
			Type:      LogTypeConsume,
			TokenId:   101,
			TokenName: "codex-key",
			ModelName: "gpt-5.5",
			Group:     "default",
			ChannelId: 22,
			Quota:     200,
		},
	}
	require.NoError(t, DB.Create(&logs).Error)

	rows, total, summary, err := GetCostStatistics(CostStatisticsQuery{
		IsAdmin:        true,
		LogType:        LogTypeConsume,
		StartTimestamp: base.Add(-time.Hour).Unix(),
		EndTimestamp:   base.Add(time.Hour).Unix(),
	}, 0, 10)
	require.NoError(t, err)

	require.Equal(t, 2, total)
	assert.Equal(t, 300, summary.Quota)
	byChannel := make(map[int]*CostStatisticsRow)
	for _, row := range rows {
		byChannel[row.ChannelId] = row
	}
	assert.Equal(t, "primary-channel", byChannel[11].ChannelName)
	assert.Equal(t, "backup-channel", byChannel[22].ChannelName)
}

func TestGetCostStatisticsAggregatesExportBillingFields(t *testing.T) {
	truncateTables(t)

	base := time.Date(2026, 7, 16, 9, 0, 0, 0, time.Local)
	logs := []*Log{
		{
			UserId:           7,
			Username:         "alice",
			CreatedAt:        base.Unix(),
			Type:             LogTypeConsume,
			TokenId:          101,
			TokenName:        "codex-key",
			ModelName:        "claude-sonnet-4-6",
			Group:            "default",
			Quota:            100,
			PromptTokens:     100,
			CompletionTokens: 50,
			Other:            `{"usage_semantic":"anthropic","claude":true,"model_ratio":3,"group_ratio":0.5,"completion_ratio":5,"cache_tokens":20,"cache_ratio":0.3,"cache_creation_tokens_5m":7,"cache_creation_ratio_5m":1.25,"cache_creation_tokens_1h":3,"cache_creation_ratio_1h":2,"model_price":0}`,
		},
		{
			UserId:           7,
			Username:         "alice",
			CreatedAt:        base.Add(time.Minute).Unix(),
			Type:             LogTypeConsume,
			TokenId:          101,
			TokenName:        "codex-key",
			ModelName:        "claude-sonnet-4-6",
			Group:            "default",
			Quota:            200,
			PromptTokens:     40,
			CompletionTokens: 10,
			Other:            `{"usage_semantic":"anthropic","claude":true,"model_ratio":3,"group_ratio":0.5,"completion_ratio":5,"cache_tokens":4,"cache_ratio":0.3,"cache_creation_tokens_5m":2,"cache_creation_ratio_5m":1.25}`,
		},
	}
	require.NoError(t, DB.Create(&logs).Error)

	rows, total, _, err := GetCostStatistics(CostStatisticsQuery{
		UserId:         7,
		IsAdmin:        false,
		LogType:        LogTypeConsume,
		StartTimestamp: base.Add(-time.Hour).Unix(),
		EndTimestamp:   base.Add(time.Hour).Unix(),
	}, 0, 10)
	require.NoError(t, err)

	require.Equal(t, 1, total)
	require.Len(t, rows, 1)
	row := rows[0]
	assert.Equal(t, "alice", row.Username)
	assert.Equal(t, 140, row.PromptTokens)
	assert.Equal(t, 140, row.InputTokens)
	assert.Equal(t, 60, row.CompletionTokens)
	assert.Equal(t, 24, row.CacheTokens)
	assert.Equal(t, 9, row.CacheWrite5m)
	assert.Equal(t, 3, row.CacheWrite1h)
	assert.Equal(t, 3.0, row.ModelRatio)
	assert.Equal(t, 0.5, row.GroupRatio)
	assert.Equal(t, 5.0, row.CompletionRatio)
	assert.Equal(t, 0.3, row.CacheRatio)
	assert.Equal(t, 1.25, row.CacheWrite5mRate)
	assert.Equal(t, 2.0, row.CacheWrite1hRate)
	assert.Equal(t, 6.0, row.InputUnitPrice)
	assert.Equal(t, 30.0, row.OutputUnitPrice)
	assert.InDelta(t, 1.8, row.CacheReadPrice, 1e-12)
	assert.Equal(t, 7.5, row.CacheWritePrice)
	assert.Equal(t, 12.0, row.CacheWrite1hPrice)
	assert.Equal(t, "按量倍率", row.BillingMode)
	assert.Empty(t, row.MatchedTier)
}

func TestGetCostStatisticsResolvesAndSeparatesTieredUnitPrices(t *testing.T) {
	truncateTables(t)

	base := time.Date(2026, 8, 3, 9, 0, 0, 0, time.Local)
	exprStr := `len <= 272000 ? tier("standard", p * 5 + c * 30 + cr * 0.5 + cc * 6.25) : tier("long_context", p * 10 + c * 45 + cr * 1 + cc * 12.5)`
	exprB64 := base64.StdEncoding.EncodeToString([]byte(exprStr))
	logs := []*Log{
		{
			UserId:           7,
			Username:         "alice",
			CreatedAt:        base.Unix(),
			Type:             LogTypeConsume,
			TokenId:          101,
			TokenName:        "codex-key",
			ModelName:        "gpt-5.6-sol",
			Group:            "default",
			Quota:            100,
			PromptTokens:     100,
			CompletionTokens: 20,
			Other:            `{"group_ratio":0.5,"cache_tokens":30,"cache_creation_tokens":10,"billing_mode":"tiered_expr","matched_tier":"standard","expr_b64":"` + exprB64 + `"}`,
		},
		{
			UserId:           7,
			Username:         "alice",
			CreatedAt:        base.Add(time.Minute).Unix(),
			Type:             LogTypeConsume,
			TokenId:          101,
			TokenName:        "codex-key",
			ModelName:        "gpt-5.6-sol",
			Group:            "default",
			Quota:            200,
			PromptTokens:     300000,
			CompletionTokens: 40,
			Other:            `{"group_ratio":0.5,"cache_tokens":50,"cache_creation_tokens":20,"billing_mode":"tiered_expr","matched_tier":"long_context","expr_b64":"` + exprB64 + `"}`,
		},
	}
	require.NoError(t, DB.Create(&logs).Error)

	rows, total, _, err := GetCostStatistics(CostStatisticsQuery{
		UserId:         7,
		IsAdmin:        false,
		LogType:        LogTypeConsume,
		StartTimestamp: base.Add(-time.Hour).Unix(),
		EndTimestamp:   base.Add(time.Hour).Unix(),
	}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 2, total)

	byTier := make(map[string]*CostStatisticsRow)
	for _, row := range rows {
		byTier[row.MatchedTier] = row
	}
	assert.Equal(t, &CostStatisticsRow{
		Date:             base.Format("20060102"),
		Type:             LogTypeConsume,
		TokenId:          101,
		TokenName:        "codex-key",
		ModelName:        "gpt-5.6-sol",
		Group:            "default",
		Username:         "alice",
		PromptTokens:     100,
		InputTokens:      60,
		CompletionTokens: 20,
		TotalTokens:      120,
		RequestCount:     1,
		Quota:            100,
		GroupRatio:       0.5,
		CacheTokens:      30,
		CacheWrite5m:     10,
		InputUnitPrice:   5,
		OutputUnitPrice:  30,
		CacheReadPrice:   0.5,
		CacheWritePrice:  6.25,
		BillingMode:      "动态计费",
		MatchedTier:      "standard",
	}, byTier["standard"])
	assert.Equal(t, 10.0, byTier["long_context"].InputUnitPrice)
	assert.Equal(t, 45.0, byTier["long_context"].OutputUnitPrice)
	assert.Equal(t, 1.0, byTier["long_context"].CacheReadPrice)
	assert.Equal(t, 12.5, byTier["long_context"].CacheWritePrice)
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

func TestGetCostStatisticsScansAllBatches(t *testing.T) {
	truncateTables(t)

	base := time.Date(2026, 7, 22, 0, 0, 0, 0, time.Local)
	logs := make([]*Log, 0, 2500)
	for i := 0; i < 2500; i++ {
		logs = append(logs, &Log{
			UserId:    7,
			Username:  "transsion",
			CreatedAt: base.Add(time.Duration(i) * time.Second).Unix(),
			Type:      LogTypeConsume,
			TokenId:   101,
			TokenName: "api-key",
			ModelName: "gpt-5.6-sol",
			Group:     "OpenAI官方",
			ChannelId: 18,
			Quota:     100,
		})
	}
	require.NoError(t, DB.CreateInBatches(&logs, 200).Error)

	rows, total, summary, err := GetCostStatistics(CostStatisticsQuery{
		IsAdmin:        true,
		LogType:        LogTypeConsume,
		StartTimestamp: base.Unix(),
		EndTimestamp:   base.Add(2499 * time.Second).Unix(),
		Username:       "transsion",
	}, 0, 10)
	require.NoError(t, err)

	require.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, 2500, summary.RequestCount)
	assert.Equal(t, 250000, summary.Quota)
	assert.Equal(t, 2500, rows[0].RequestCount)
	assert.Equal(t, 250000, rows[0].Quota)
}
