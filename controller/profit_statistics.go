package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func GetProfitStatistics(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	startTimestamp, endTimestamp, period, err := resolveCostStatisticsPeriod(
		c.Query("period"),
		true,
		c.Query("start_timestamp"),
		c.Query("end_timestamp"),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	channelId, _ := strconv.Atoi(c.Query("channel"))
	modelName := strings.TrimSpace(c.Query("model_name"))
	if len(modelName) >= 2 && !strings.Contains(modelName, "%") {
		modelName = "%" + modelName + "%"
	}
	query := model.ProfitStatisticsQuery{
		LogType:        parseCostStatisticsLogType(c.Query("type")),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		Username:       c.Query("username"),
		ModelName:      modelName,
		ChannelId:      channelId,
		Group:          c.Query("group"),
	}
	rows, total, summary, err := model.GetProfitStatistics(query, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
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

func ListChannelProfitRatios(c *gin.Context) {
	ratios, err := model.ListChannelProfitRatios()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, ratios)
}

func SaveChannelProfitRatio(c *gin.Context) {
	var input model.ChannelProfitRatioInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	ratio, err := model.SaveChannelProfitRatio(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, ratio)
}

func DeleteChannelProfitRatio(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteChannelProfitRatio(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
