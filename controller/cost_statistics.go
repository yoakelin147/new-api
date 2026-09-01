package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

const (
	costStatisticsPeriodToday                         = "today"
	costStatisticsPeriodYesterday                     = "yesterday"
	costStatisticsPeriodWeek                          = "week"
	costStatisticsPeriodMonth                         = "month"
	costStatisticsPeriodLastMonth                     = "last_month"
	costStatisticsPeriodCustom                        = "custom"
	maxNonAdminCostStatisticsCustomRangeSeconds int64 = 31 * 24 * 60 * 60
)

func GetCostStatistics(c *gin.Context) {
	handleCostStatistics(c, true)
}

func GetSelfCostStatistics(c *gin.Context) {
	handleCostStatistics(c, false)
}

func GetManagedCostStatistics(c *gin.Context) {
	userIds, ok := getManagedLogUserIds(c)
	if !ok {
		return
	}
	if userIds == nil {
		userIds = []int{}
	}
	handleCostStatisticsForUsers(c, userIds)
}

func handleCostStatistics(c *gin.Context, isAdmin bool) {
	handleCostStatisticsWithQuery(c, isAdmin, nil)
}

func handleCostStatisticsForUsers(c *gin.Context, userIds []int) {
	handleCostStatisticsWithQuery(c, false, userIds)
}

func handleCostStatisticsWithQuery(c *gin.Context, isAdmin bool, userIds []int) {
	pageInfo := common.GetPageQuery(c)
	exportAll := c.Query("export") == "true"
	startTimestamp, endTimestamp, period, err := resolveCostStatisticsPeriod(
		c.Query("period"),
		isAdmin,
		c.Query("start_timestamp"),
		c.Query("end_timestamp"),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	logType := parseCostStatisticsLogType(c.Query("type"))
	channelId, _ := strconv.Atoi(c.Query("channel"))

	modelName := strings.TrimSpace(c.Query("model_name"))
	if len(modelName) >= 2 && !strings.Contains(modelName, "%") {
		modelName = "%" + modelName + "%"
	}

	query := model.CostStatisticsQuery{
		UserId:         c.GetInt("id"),
		UserIds:        userIds,
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

	query.ExportAll = exportAll
	startIdx := pageInfo.GetStartIdx()
	pageSize := pageInfo.GetPageSize()
	if exportAll {
		startIdx = 0
		pageSize = 0
	}
	rows, total, summary, err := model.GetCostStatistics(query, startIdx, pageSize)
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

func resolveCostStatisticsPeriod(raw string, isAdmin bool, rawStart string, rawEnd string) (int64, int64, string, error) {
	now := time.Now()
	period := raw
	var start time.Time
	var end time.Time
	switch period {
	case costStatisticsPeriodYesterday:
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		start = todayStart.AddDate(0, 0, -1)
		end = todayStart.Add(-time.Second)
	case costStatisticsPeriodWeek:
		offset := (int(now.Weekday()) + 6) % 7
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -offset)
		end = now
	case costStatisticsPeriodMonth:
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = now
	case costStatisticsPeriodLastMonth:
		thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		start = thisMonthStart.AddDate(0, -1, 0)
		end = thisMonthStart.Add(-time.Second)
	case costStatisticsPeriodCustom:
		startTimestamp, err := strconv.ParseInt(rawStart, 10, 64)
		if err != nil || startTimestamp <= 0 {
			return 0, 0, "", errors.New("invalid start time")
		}
		endTimestamp, err := strconv.ParseInt(rawEnd, 10, 64)
		if err != nil || endTimestamp <= 0 {
			return 0, 0, "", errors.New("invalid end time")
		}
		if endTimestamp < startTimestamp {
			return 0, 0, "", errors.New("end time must be later than start time")
		}
		if !isAdmin && endTimestamp-startTimestamp > maxNonAdminCostStatisticsCustomRangeSeconds {
			return 0, 0, "", errors.New("custom time range cannot exceed 31 days")
		}
		return startTimestamp, endTimestamp, period, nil
	default:
		period = costStatisticsPeriodToday
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = now
	}
	return start.Unix(), end.Unix(), period, nil
}
