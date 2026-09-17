package model

import (
	"encoding/base64"
	"errors"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
)

type CostStatisticsQuery struct {
	ReadState *LogReadState
	UserId    int
	// UserIds scopes a non-admin query to an explicit set of users. Nil keeps
	// the legacy single-user behavior; an empty non-nil slice matches no logs.
	UserIds        []int
	IsAdmin        bool
	LogType        int
	StartTimestamp int64
	EndTimestamp   int64
	Username       string
	TokenName      string
	ModelName      string
	ChannelId      int
	Group          string
	// ExportAll disables result pagination for the authenticated export path.
	// Summary aggregation is unchanged; only the returned rows are unpaged.
	ExportAll bool
}

type CostStatisticsSummary struct {
	Quota            int `json:"quota"`
	RequestCount     int `json:"request_count"`
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type CostStatisticsRow struct {
	Date              string  `json:"date"`
	Type              int     `json:"type"`
	TokenId           int     `json:"token_id"`
	TokenName         string  `json:"token_name"`
	TokenKeyMasked    string  `json:"token_key_masked"`
	ModelName         string  `json:"model_name"`
	Group             string  `json:"group"`
	Username          string  `json:"username"`
	ChannelId         int     `json:"channel_id"`
	ChannelName       string  `json:"channel_name"`
	PromptTokens      int     `json:"prompt_tokens"`
	InputTokens       int     `json:"input_tokens"`
	CompletionTokens  int     `json:"completion_tokens"`
	TotalTokens       int     `json:"total_tokens"`
	RequestCount      int     `json:"request_count"`
	Quota             int     `json:"quota"`
	GroupRatio        float64 `json:"group_ratio"`
	ModelRatio        float64 `json:"model_ratio"`
	CompletionRatio   float64 `json:"completion_ratio"`
	CacheTokens       int     `json:"cache_tokens"`
	CacheRatio        float64 `json:"cache_ratio"`
	CacheWrite5m      int     `json:"cache_write_5m_tokens"`
	CacheWrite5mRate  float64 `json:"cache_write_5m_ratio"`
	CacheWrite1h      int     `json:"cache_write_1h_tokens"`
	CacheWrite1hRate  float64 `json:"cache_write_1h_ratio"`
	InputUnitPrice    float64 `json:"input_unit_price"`
	OutputUnitPrice   float64 `json:"output_unit_price"`
	CacheReadPrice    float64 `json:"cache_read_unit_price"`
	CacheWritePrice   float64 `json:"cache_write_unit_price"`
	CacheWrite1hPrice float64 `json:"cache_write_1h_unit_price"`
	ModelPrice        float64 `json:"model_price"`
	BillingMode       string  `json:"billing_mode"`
	MatchedTier       string  `json:"matched_tier"`
	UseTimeSeconds    int     `json:"use_time_seconds"`
}

type costStatisticsKey struct {
	Date           string
	Type           int
	TokenId        int
	TokenName      string
	TokenKeyMasked string
	ModelName      string
	Group          string
	Username       string
	ChannelId      int
	ChannelName    string
	Pricing        costStatisticsPricing
}

type costStatisticsPricing struct {
	GroupRatio          float64
	ModelRatio          float64
	CompletionRatio     float64
	CacheRatio          float64
	CacheWriteRate      float64
	CacheWrite1hRate    float64
	ModelPrice          float64
	BillingMode         string
	MatchedTier         string
	DynamicExprIdentity string
	InputUnitPrice      float64
	OutputUnitPrice     float64
	CacheReadPrice      float64
	CacheWritePrice     float64
	CacheWrite1hPrice   float64
}

type costStatisticsLogOther struct {
	TokenAudit            *costStatisticsTokenAudit `json:"token_audit"`
	ModelRatio            float64                   `json:"model_ratio"`
	GroupRatio            float64                   `json:"group_ratio"`
	CompletionRatio       float64                   `json:"completion_ratio"`
	CacheTokens           int                       `json:"cache_tokens"`
	CacheRatio            float64                   `json:"cache_ratio"`
	CacheCreationTokens   int                       `json:"cache_creation_tokens"`
	CacheCreationRatio    float64                   `json:"cache_creation_ratio"`
	CacheCreationTokens5m int                       `json:"cache_creation_tokens_5m"`
	CacheCreationRatio5m  float64                   `json:"cache_creation_ratio_5m"`
	CacheCreationTokens1h int                       `json:"cache_creation_tokens_1h"`
	CacheCreationRatio1h  float64                   `json:"cache_creation_ratio_1h"`
	ModelPrice            float64                   `json:"model_price"`
	BillingMode           string                    `json:"billing_mode"`
	MatchedTier           string                    `json:"matched_tier"`
	ExprB64               string                    `json:"expr_b64"`
	UnitPricesResolved    bool                      `json:"unit_prices_resolved"`
	InputUnitPrice        float64                   `json:"input_unit_price"`
	OutputUnitPrice       float64                   `json:"output_unit_price"`
	CacheReadUnitPrice    float64                   `json:"cache_read_unit_price"`
	CacheWriteUnitPrice   float64                   `json:"cache_write_unit_price"`
	CacheWrite1hUnitPrice float64                   `json:"cache_write_1h_unit_price"`
	UsageSemantic         string                    `json:"usage_semantic"`
	Claude                bool                      `json:"claude"`
}

type costStatisticsTokenAudit struct {
	MaskedKey string `json:"masked_key"`
}

func GetCostStatistics(query CostStatisticsQuery, startIdx int, pageSize int) ([]*CostStatisticsRow, int, CostStatisticsSummary, error) {
	if query.ReadState == nil {
		query.ReadState = NewLogReadState(nil)
	}
	rowsByKey := make(map[costStatisticsKey]*CostStatisticsRow)
	channelIds := make(map[int]struct{})
	summary := CostStatisticsSummary{}
	if !query.IsAdmin && query.UserIds != nil && len(query.UserIds) == 0 {
		return []*CostStatisticsRow{}, 0, summary, nil
	}
	if query.IsAdmin {
		if err := validateLogTextPatterns(query.Username); err != nil {
			return nil, 0, summary, err
		}
	}
	if err := validateLogTextPatterns(query.ModelName); err != nil {
		return nil, 0, summary, err
	}

	err := forEachCostLog(query, func(log Log) error {
		date := time.Unix(log.CreatedAt, 0).Local().Format("20060102")
		other := parseCostStatisticsLogOther(log.Other)
		pricing := resolveCostStatisticsPricing(other)
		channelId := 0
		if query.IsAdmin {
			channelId = log.ChannelId
		}
		key := costStatisticsKey{
			Date:           date,
			Type:           log.Type,
			TokenId:        log.TokenId,
			TokenName:      log.TokenName,
			TokenKeyMasked: tokenAuditMaskedKey(other),
			ModelName:      log.ModelName,
			Group:          log.Group,
			Username:       log.Username,
			ChannelId:      channelId,
			Pricing:        pricing,
		}
		row := rowsByKey[key]
		if row == nil {
			row = &CostStatisticsRow{
				Date:              date,
				Type:              log.Type,
				TokenId:           log.TokenId,
				TokenName:         log.TokenName,
				TokenKeyMasked:    tokenAuditMaskedKey(other),
				ModelName:         log.ModelName,
				Group:             log.Group,
				Username:          log.Username,
				ChannelId:         channelId,
				GroupRatio:        pricing.GroupRatio,
				ModelRatio:        pricing.ModelRatio,
				CompletionRatio:   pricing.CompletionRatio,
				CacheRatio:        pricing.CacheRatio,
				CacheWrite5mRate:  pricing.CacheWriteRate,
				CacheWrite1hRate:  pricing.CacheWrite1hRate,
				InputUnitPrice:    pricing.InputUnitPrice,
				OutputUnitPrice:   pricing.OutputUnitPrice,
				CacheReadPrice:    pricing.CacheReadPrice,
				CacheWritePrice:   pricing.CacheWritePrice,
				CacheWrite1hPrice: pricing.CacheWrite1hPrice,
				ModelPrice:        pricing.ModelPrice,
				BillingMode:       pricing.BillingMode,
				MatchedTier:       pricing.MatchedTier,
			}
			rowsByKey[key] = row
		}
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
		summary.PromptTokens += log.PromptTokens
		summary.CompletionTokens += log.CompletionTokens
		summary.TotalTokens += log.PromptTokens + log.CompletionTokens
		summary.RequestCount++
		summary.Quota += quota
		if query.IsAdmin && log.ChannelId != 0 {
			channelIds[log.ChannelId] = struct{}{}
		}
		return nil
	})
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
	if query.ExportAll {
		return rows, total, summary, nil
	}
	if startIdx >= total {
		return []*CostStatisticsRow{}, total, summary, nil
	}
	endIdx := startIdx + pageSize
	if endIdx > total {
		endIdx = total
	}
	return rows[startIdx:endIdx], total, summary, nil
}

func tokenAuditMaskedKey(other costStatisticsLogOther) string {
	if other.TokenAudit == nil {
		return ""
	}
	return other.TokenAudit.MaskedKey
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

func resolveCostStatisticsPricing(other costStatisticsLogOther) costStatisticsPricing {
	cacheWriteRate := other.CacheCreationRatio5m
	if cacheWriteRate == 0 {
		cacheWriteRate = other.CacheCreationRatio
	}
	cacheWrite1hRate := other.CacheCreationRatio1h
	if cacheWrite1hRate == 0 && (other.UsageSemantic == "anthropic" || other.Claude) && cacheWriteRate != 0 {
		cacheWrite1hRate = cacheWriteRate * 1.6
	}
	inputPrice, outputPrice, cacheReadPrice, cacheWritePrice, cacheWrite1hPrice := costStatisticsUnitPrices(other, cacheWriteRate, cacheWrite1hRate)
	dynamicExprIdentity := ""
	if other.BillingMode == "tiered_expr" {
		dynamicExprIdentity = other.ExprB64
	}
	return costStatisticsPricing{
		GroupRatio:          other.GroupRatio,
		ModelRatio:          other.ModelRatio,
		CompletionRatio:     other.CompletionRatio,
		CacheRatio:          other.CacheRatio,
		CacheWriteRate:      cacheWriteRate,
		CacheWrite1hRate:    cacheWrite1hRate,
		ModelPrice:          other.ModelPrice,
		BillingMode:         costStatisticsBillingMode(other),
		MatchedTier:         other.MatchedTier,
		DynamicExprIdentity: dynamicExprIdentity,
		InputUnitPrice:      inputPrice,
		OutputUnitPrice:     outputPrice,
		CacheReadPrice:      cacheReadPrice,
		CacheWritePrice:     cacheWritePrice,
		CacheWrite1hPrice:   cacheWrite1hPrice,
	}
}

func costStatisticsUnitPrices(other costStatisticsLogOther, cacheWriteRate float64, cacheWrite1hRate float64) (float64, float64, float64, float64, float64) {
	if other.BillingMode == "tiered_expr" {
		if other.UnitPricesResolved {
			return other.InputUnitPrice, other.OutputUnitPrice, other.CacheReadUnitPrice, other.CacheWriteUnitPrice, other.CacheWrite1hUnitPrice
		}
		exprBytes, err := base64.StdEncoding.DecodeString(other.ExprB64)
		if err != nil {
			return 0, 0, 0, 0, 0
		}
		prices, ok := billingexpr.ExtractTierUnitPrices(string(exprBytes), other.MatchedTier)
		if !ok {
			return 0, 0, 0, 0, 0
		}
		return prices.Input, prices.Output, prices.CacheRead, prices.CacheWrite, prices.CacheWrite1h
	}
	if other.ModelPrice > 0 {
		return 0, 0, 0, 0, 0
	}
	inputPrice := other.ModelRatio * 2
	return inputPrice,
		inputPrice * other.CompletionRatio,
		inputPrice * other.CacheRatio,
		inputPrice * cacheWriteRate,
		inputPrice * cacheWrite1hRate
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
