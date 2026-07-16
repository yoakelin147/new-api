package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

const (
	costStatisticsPeriodToday = "today"
	costStatisticsPeriodWeek  = "week"
	costStatisticsPeriodMonth = "month"
)

func GetCostStatistics(c *gin.Context) {
	handleCostStatistics(c, true)
}

func GetSelfCostStatistics(c *gin.Context) {
	handleCostStatistics(c, false)
}

func handleCostStatistics(c *gin.Context, isAdmin bool) {
	pageInfo := common.GetPageQuery(c)
	startTimestamp, endTimestamp, period := resolveCostStatisticsPeriod(c.Query("period"))
	logType := parseCostStatisticsLogType(c.Query("type"))
	channelId, _ := strconv.Atoi(c.Query("channel"))

	modelName := strings.TrimSpace(c.Query("model_name"))
	if len(modelName) >= 2 && !strings.Contains(modelName, "%") {
		modelName = "%" + modelName + "%"
	}

	query := model.CostStatisticsQuery{
		UserId:         c.GetInt("id"),
		IsAdmin:        isAdmin,
		LogType:        logType,
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		Username:       c.Query("username"),
		TokenName:      c.Query("token_name"),
		ModelName:      modelName,
		ChannelId:      channelId,
		Group:          c.Query("group"),
	}

	rows, total, summary, err := model.GetCostStatistics(query, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(total)
	pageInfo.SetItems(rows)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"page":            pageInfo.Page,
			"page_size":       pageInfo.PageSize,
			"total":           pageInfo.Total,
			"items":           pageInfo.Items,
			"summary":         summary,
			"period":          period,
			"start_timestamp": startTimestamp,
			"end_timestamp":   endTimestamp,
		},
	})
}

func parseCostStatisticsLogType(raw string) int {
	switch raw {
	case "consume":
		return model.LogTypeConsume
	case "refund":
		return model.LogTypeRefund
	default:
		return model.LogTypeUnknown
	}
}

func resolveCostStatisticsPeriod(raw string) (int64, int64, string) {
	now := time.Now()
	period := raw
	var start time.Time
	switch period {
	case costStatisticsPeriodWeek:
		offset := (int(now.Weekday()) + 6) % 7
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -offset)
	case costStatisticsPeriodMonth:
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		period = costStatisticsPeriodToday
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}
	return start.Unix(), now.Unix(), period
}
