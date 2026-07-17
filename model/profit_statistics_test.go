package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProfitStatisticsCalculatesProfitByRatios(t *testing.T) {
	truncateTables(t)
	common.QuotaPerUnit = 500000

	base := time.Date(2026, 7, 17, 10, 0, 0, 0, time.Local)
	require.NoError(t, DB.Create(&Channel{Id: 18, Name: "Azure", Key: "key"}).Error)
	require.NoError(t, DB.Create(&ChannelProfitRatio{
		ChannelId:     18,
		UpstreamRatio: 0.3,
		Enabled:       true,
	}).Error)
	require.NoError(t, DB.Create(&Log{
		UserId:           7,
		Username:         "alice",
		CreatedAt:        base.Unix(),
		Type:             LogTypeConsume,
		ModelName:        "gpt-5.5",
		Group:            "default",
		ChannelId:        18,
		Quota:            int(100 * common.QuotaPerUnit),
		PromptTokens:     100,
		CompletionTokens: 50,
		Other:            `{"group_ratio":0.5}`,
	}).Error)

	rows, total, summary, err := GetProfitStatistics(ProfitStatisticsQuery{
		LogType:        LogTypeConsume,
		StartTimestamp: base.Add(-time.Hour).Unix(),
		EndTimestamp:   base.Add(time.Hour).Unix(),
	}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, rows, 1)

	row := rows[0]
	require.NotNil(t, row.UpstreamRatio)
	require.NotNil(t, row.UpstreamCost)
	require.NotNil(t, row.Profit)
	require.NotNil(t, row.ProfitRate)
	assert.Equal(t, "Azure", row.ChannelName)
	assert.Equal(t, 0.5, row.CustomerRatio)
	assert.Equal(t, 0.3, *row.UpstreamRatio)
	assert.InDelta(t, 100, row.CustomerAmount, 0.000001)
	assert.InDelta(t, 200, row.OriginalAmount, 0.000001)
	assert.InDelta(t, 60, *row.UpstreamCost, 0.000001)
	assert.InDelta(t, 40, *row.Profit, 0.000001)
	assert.InDelta(t, 0.4, *row.ProfitRate, 0.000001)
	assert.Equal(t, ProfitCostStatusConfigured, row.CostStatus)
	assert.InDelta(t, 60, summary.UpstreamCost, 0.000001)
	assert.InDelta(t, 40, summary.Profit, 0.000001)
	assert.InDelta(t, 0.4, summary.ProfitRate, 0.000001)
}

func TestGetProfitStatisticsMarksUnconfiguredCost(t *testing.T) {
	truncateTables(t)
	common.QuotaPerUnit = 500000

	base := time.Date(2026, 7, 17, 10, 0, 0, 0, time.Local)
	require.NoError(t, DB.Create(&Channel{Id: 18, Name: "Azure", Key: "key"}).Error)
	require.NoError(t, DB.Create(&Log{
		UserId:    7,
		Username:  "alice",
		CreatedAt: base.Unix(),
		Type:      LogTypeConsume,
		ModelName: "gpt-5.5",
		Group:     "default",
		ChannelId: 18,
		Quota:     int(100 * common.QuotaPerUnit),
		Other:     `{"group_ratio":0.5}`,
	}).Error)

	rows, total, summary, err := GetProfitStatistics(ProfitStatisticsQuery{
		LogType:        LogTypeConsume,
		StartTimestamp: base.Add(-time.Hour).Unix(),
		EndTimestamp:   base.Add(time.Hour).Unix(),
	}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 1, total)

	row := rows[0]
	assert.Nil(t, row.UpstreamRatio)
	assert.Nil(t, row.UpstreamCost)
	assert.Nil(t, row.Profit)
	assert.Equal(t, ProfitCostStatusUnconfigured, row.CostStatus)
	assert.Equal(t, 1, summary.UnconfiguredRows)
	assert.InDelta(t, 100, summary.UnconfiguredAmount, 0.000001)
}

func TestGetProfitStatisticsSummaryProfitRateIgnoresUnconfiguredAmount(t *testing.T) {
	truncateTables(t)
	common.QuotaPerUnit = 500000

	base := time.Date(2026, 7, 17, 10, 0, 0, 0, time.Local)
	require.NoError(t, DB.Create(&Channel{Id: 18, Name: "Configured", Key: "key"}).Error)
	require.NoError(t, DB.Create(&Channel{Id: 19, Name: "Unconfigured", Key: "key"}).Error)
	require.NoError(t, DB.Create(&ChannelProfitRatio{
		ChannelId:     18,
		UpstreamRatio: 0.3,
		Enabled:       true,
	}).Error)
	require.NoError(t, DB.Create(&Log{
		UserId:    7,
		Username:  "alice",
		CreatedAt: base.Unix(),
		Type:      LogTypeConsume,
		ModelName: "gpt-5.5",
		Group:     "default",
		ChannelId: 18,
		Quota:     int(100 * common.QuotaPerUnit),
		Other:     `{"group_ratio":0.5}`,
	}).Error)
	require.NoError(t, DB.Create(&Log{
		UserId:    7,
		Username:  "alice",
		CreatedAt: base.Unix(),
		Type:      LogTypeConsume,
		ModelName: "gpt-5.5",
		Group:     "default",
		ChannelId: 19,
		Quota:     int(100 * common.QuotaPerUnit),
		Other:     `{"group_ratio":0.5}`,
	}).Error)

	_, _, summary, err := GetProfitStatistics(ProfitStatisticsQuery{
		LogType:        LogTypeConsume,
		StartTimestamp: base.Add(-time.Hour).Unix(),
		EndTimestamp:   base.Add(time.Hour).Unix(),
	}, 0, 10)
	require.NoError(t, err)

	assert.InDelta(t, 200, summary.CustomerAmount, 0.000001)
	assert.InDelta(t, 100, summary.UnconfiguredAmount, 0.000001)
	assert.InDelta(t, 40, summary.Profit, 0.000001)
	assert.InDelta(t, 0.4, summary.ProfitRate, 0.000001)
}

func TestGetProfitStatisticsSeparatesUsersByDefault(t *testing.T) {
	truncateTables(t)
	common.QuotaPerUnit = 500000

	base := time.Date(2026, 7, 17, 10, 0, 0, 0, time.Local)
	require.NoError(t, DB.Create(&Channel{Id: 18, Name: "Azure", Key: "key"}).Error)
	require.NoError(t, DB.Create(&ChannelProfitRatio{
		ChannelId:     18,
		UpstreamRatio: 0.3,
		Enabled:       true,
	}).Error)
	for _, username := range []string{"alice", "bob"} {
		require.NoError(t, DB.Create(&Log{
			UserId:    7,
			Username:  username,
			CreatedAt: base.Unix(),
			Type:      LogTypeConsume,
			ModelName: "gpt-5.5",
			Group:     "default",
			ChannelId: 18,
			Quota:     int(100 * common.QuotaPerUnit),
			Other:     `{"group_ratio":0.5}`,
		}).Error)
	}

	rows, total, _, err := GetProfitStatistics(ProfitStatisticsQuery{
		LogType:        LogTypeConsume,
		StartTimestamp: base.Add(-time.Hour).Unix(),
		EndTimestamp:   base.Add(time.Hour).Unix(),
	}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, rows, 2)

	assert.Equal(t, "alice", rows[0].Username)
	assert.Equal(t, 1, rows[0].RequestCount)
	assert.InDelta(t, 100, rows[0].CustomerAmount, 0.000001)
	assert.Equal(t, "bob", rows[1].Username)
	assert.Equal(t, 1, rows[1].RequestCount)
	assert.InDelta(t, 100, rows[1].CustomerAmount, 0.000001)
}

func TestGetProfitStatisticsShowsUsernameWhenFiltered(t *testing.T) {
	truncateTables(t)
	common.QuotaPerUnit = 500000

	base := time.Date(2026, 7, 17, 10, 0, 0, 0, time.Local)
	require.NoError(t, DB.Create(&Channel{Id: 18, Name: "Azure", Key: "key"}).Error)
	require.NoError(t, DB.Create(&ChannelProfitRatio{
		ChannelId:     18,
		UpstreamRatio: 0.3,
		Enabled:       true,
	}).Error)
	for _, username := range []string{"alice", "bob"} {
		require.NoError(t, DB.Create(&Log{
			UserId:    7,
			Username:  username,
			CreatedAt: base.Unix(),
			Type:      LogTypeConsume,
			ModelName: "gpt-5.5",
			Group:     "default",
			ChannelId: 18,
			Quota:     int(100 * common.QuotaPerUnit),
			Other:     `{"group_ratio":0.5}`,
		}).Error)
	}

	rows, total, _, err := GetProfitStatistics(ProfitStatisticsQuery{
		LogType:        LogTypeConsume,
		StartTimestamp: base.Add(-time.Hour).Unix(),
		EndTimestamp:   base.Add(time.Hour).Unix(),
		Username:       "alice",
	}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 1, total)

	assert.Equal(t, "alice", rows[0].Username)
	assert.Equal(t, 1, rows[0].RequestCount)
	assert.InDelta(t, 100, rows[0].CustomerAmount, 0.000001)
}

func TestGetProfitStatisticsSeparatesChangedCustomerRatios(t *testing.T) {
	truncateTables(t)
	common.QuotaPerUnit = 500000

	base := time.Date(2026, 7, 17, 10, 0, 0, 0, time.Local)
	require.NoError(t, DB.Create(&Channel{Id: 18, Name: "Claude", Key: "key"}).Error)
	require.NoError(t, DB.Create(&ChannelProfitRatio{
		ChannelId:     18,
		UpstreamRatio: 0.4,
		Enabled:       true,
	}).Error)
	for _, item := range []struct {
		ratio string
		quota int
	}{
		{ratio: `{"group_ratio":0.4}`, quota: int(40 * common.QuotaPerUnit)},
		{ratio: `{"group_ratio":4}`, quota: int(80 * common.QuotaPerUnit)},
	} {
		require.NoError(t, DB.Create(&Log{
			UserId:    7,
			Username:  "alice",
			CreatedAt: base.Unix(),
			Type:      LogTypeConsume,
			ModelName: "claude-sonnet-4-6",
			Group:     "Claude Code专属",
			ChannelId: 18,
			Quota:     item.quota,
			Other:     item.ratio,
		}).Error)
	}

	rows, total, summary, err := GetProfitStatistics(ProfitStatisticsQuery{
		LogType:        LogTypeConsume,
		StartTimestamp: base.Add(-time.Hour).Unix(),
		EndTimestamp:   base.Add(time.Hour).Unix(),
		Username:       "alice",
	}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, rows, 2)

	rowsByRatio := map[float64]*ProfitStatisticsRow{}
	for _, row := range rows {
		rowsByRatio[row.CustomerRatio] = row
	}
	require.Contains(t, rowsByRatio, 0.4)
	require.Contains(t, rowsByRatio, 4.0)

	assert.InDelta(t, 40, rowsByRatio[0.4].CustomerAmount, 0.000001)
	assert.InDelta(t, 100, rowsByRatio[0.4].OriginalAmount, 0.000001)
	require.NotNil(t, rowsByRatio[0.4].Profit)
	assert.InDelta(t, 0, *rowsByRatio[0.4].Profit, 0.000001)

	assert.InDelta(t, 80, rowsByRatio[4.0].CustomerAmount, 0.000001)
	assert.InDelta(t, 20, rowsByRatio[4.0].OriginalAmount, 0.000001)
	require.NotNil(t, rowsByRatio[4.0].Profit)
	assert.InDelta(t, 72, *rowsByRatio[4.0].Profit, 0.000001)

	assert.InDelta(t, 120, summary.CustomerAmount, 0.000001)
	assert.InDelta(t, 72, summary.Profit, 0.000001)
}

func TestChannelProfitRatioSpecificMatchWins(t *testing.T) {
	ratios := []ChannelProfitRatio{
		{Id: 1, ChannelId: 18, UpstreamRatio: 0.3},
		{Id: 2, ChannelId: 18, ModelName: "gpt-5.5", UpstreamRatio: 0.2},
		{Id: 3, ChannelId: 18, ModelName: "gpt-5.5", Group: "vip", UpstreamRatio: 0.1},
	}

	sortChannelRatiosForTest(ratios)
	ratio, ok := matchChannelProfitRatio(ratios, "gpt-5.5", "vip")
	require.True(t, ok)
	assert.Equal(t, 0.1, ratio.UpstreamRatio)

	ratio, ok = matchChannelProfitRatio(ratios, "gpt-5.5", "default")
	require.True(t, ok)
	assert.Equal(t, 0.2, ratio.UpstreamRatio)

	ratio, ok = matchChannelProfitRatio(ratios, "gpt-4", "default")
	require.True(t, ok)
	assert.Equal(t, 0.3, ratio.UpstreamRatio)
}

func sortChannelRatiosForTest(ratios []ChannelProfitRatio) {
	for i := 0; i < len(ratios)-1; i++ {
		for j := i + 1; j < len(ratios); j++ {
			if channelProfitRatioScore(ratios[j]) > channelProfitRatioScore(ratios[i]) {
				ratios[i], ratios[j] = ratios[j], ratios[i]
			}
		}
	}
}
