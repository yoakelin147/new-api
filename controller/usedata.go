package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

const maxDashboardChannelStatsRangeSeconds = 31 * 24 * 60 * 60
const maxDashboardAdminTokenStatsRangeSeconds = 183 * 24 * 60 * 60

func GetAllQuotaDates(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	username := c.Query("username")
	dates, err := model.GetAllQuotaDates(startTimestamp, endTimestamp, username)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
	return
}

func GetQuotaDatesByUser(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	dates, err := model.GetQuotaDataGroupByUser(startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
}

func GetQuotaDatesByChannel(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if endTimestamp == 0 {
		endTimestamp = time.Now().Unix()
	}
	if startTimestamp == 0 {
		startTimestamp = endTimestamp - maxDashboardChannelStatsRangeSeconds
	}
	if endTimestamp < startTimestamp {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "end_timestamp must be greater than or equal to start_timestamp",
		})
		return
	}
	if endTimestamp-startTimestamp > maxDashboardChannelStatsRangeSeconds {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "time range cannot exceed 31 days",
		})
		return
	}
	dates, err := model.GetQuotaDataGroupByChannel(startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
}

func GetQuotaDatesByToken(c *gin.Context) {
	userId := c.GetInt("id")
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if endTimestamp == 0 {
		endTimestamp = time.Now().Unix()
	}
	if startTimestamp == 0 {
		startTimestamp = endTimestamp - maxDashboardChannelStatsRangeSeconds
	}
	if endTimestamp < startTimestamp {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "end_timestamp must be greater than or equal to start_timestamp",
		})
		return
	}
	if endTimestamp-startTimestamp > maxDashboardChannelStatsRangeSeconds {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "time range cannot exceed 31 days",
		})
		return
	}
	dates, err := model.GetQuotaDataGroupByToken(userId, startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
}

func GetQuotaDatesByUserToken(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Query("user_id"))
	if userId <= 0 {
		userId = c.GetInt("id")
	}
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if endTimestamp == 0 {
		endTimestamp = time.Now().Unix()
	}
	if startTimestamp == 0 {
		startTimestamp = endTimestamp - maxDashboardChannelStatsRangeSeconds
	}
	if endTimestamp < startTimestamp {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "end_timestamp must be greater than or equal to start_timestamp",
		})
		return
	}
	if endTimestamp-startTimestamp > maxDashboardAdminTokenStatsRangeSeconds {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "time range cannot exceed 183 days",
		})
		return
	}
	dates, err := model.GetQuotaDataGroupByUserToken(userId, startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
}

func GetTokenCacheHitStats(c *gin.Context) {
	params, ok := parseTokenCacheHitStatsQuery(c)
	if !ok {
		return
	}

	stats, err := model.GetTokenCacheHitStats(params)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, stats)
}

func GetTokenCacheHitTrendStats(c *gin.Context) {
	params, ok := parseTokenCacheHitStatsQuery(c)
	if !ok {
		return
	}

	stats, err := model.GetTokenCacheHitTrendStats(params)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, stats)
}

func parseTokenCacheHitStatsQuery(c *gin.Context) (model.TokenCacheHitStatsQuery, bool) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if endTimestamp == 0 {
		endTimestamp = time.Now().Unix()
	}
	if startTimestamp == 0 {
		startTimestamp = endTimestamp - maxDashboardAdminTokenStatsRangeSeconds
	}
	if endTimestamp < startTimestamp {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "end_timestamp must be greater than or equal to start_timestamp",
		})
		return model.TokenCacheHitStatsQuery{}, false
	}
	if endTimestamp-startTimestamp > maxDashboardAdminTokenStatsRangeSeconds {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "time range cannot exceed 183 days",
		})
		return model.TokenCacheHitStatsQuery{}, false
	}

	channelId, _ := strconv.Atoi(c.Query("channel_id"))
	return model.TokenCacheHitStatsQuery{
		StartTimestamp:  startTimestamp,
		EndTimestamp:    endTimestamp,
		ModelName:       c.Query("model_name"),
		ChannelId:       channelId,
		TimeGranularity: c.Query("time_granularity"),
		TrendGroup:      c.Query("trend_group"),
	}, true
}

func GetUserQuotaDates(c *gin.Context) {
	userId := c.GetInt("id")
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	// 判断时间跨度是否超过 1 个月
	if endTimestamp-startTimestamp > 2592000 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "时间跨度不能超过 1 个月",
		})
		return
	}
	dates, err := model.GetQuotaDataByUserId(userId, startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
	return
}
