package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

const usageLogExportBatchSize = 10_000

func GetAllLogs(c *gin.Context) {
	readState := model.NewLogReadState(c.Request.Context())
	pageInfo := common.GetPageQuery(c)
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	username := c.Query("username")
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	requestId := c.Query("request_id")
	upstreamRequestId := c.Query("upstream_request_id")
	streamStatus := c.Query("stream_status")
	logs, total, err := model.GetAllLogs(logType, startTimestamp, endTimestamp, modelName, username, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), channel, group, requestId, upstreamRequestId, streamStatus, readState)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	c.Header("X-Log-Source", readState.Source())
	common.ApiSuccess(c, pageInfo)
	return
}

func ExportAllLogs(c *gin.Context) {
	readState := model.NewLogReadState(c.Request.Context())
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	channel, _ := strconv.Atoi(c.Query("channel"))
	var logs []*model.Log
	for offset := 0; ; offset += usageLogExportBatchSize {
		batch, total, err := model.GetAllLogs(
			logType,
			startTimestamp,
			endTimestamp,
			c.Query("model_name"),
			c.Query("username"),
			c.Query("token_name"),
			offset,
			usageLogExportBatchSize,
			channel,
			c.Query("group"),
			c.Query("request_id"),
			c.Query("upstream_request_id"),
			c.Query("stream_status"),
			readState,
		)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		logs = append(logs, batch...)
		if int64(len(logs)) >= total || len(batch) < usageLogExportBatchSize {
			break
		}
	}
	c.Header("X-Log-Source", readState.Source())
	common.ApiSuccess(c, logs)
}

func GetUserLogs(c *gin.Context) {
	readState := model.NewLogReadState(c.Request.Context())
	pageInfo := common.GetPageQuery(c)
	userId := c.GetInt("id")
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	group := c.Query("group")
	requestId := c.Query("request_id")
	upstreamRequestId := c.Query("upstream_request_id")
	logs, total, err := model.GetUserLogs(userId, logType, startTimestamp, endTimestamp, modelName, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), group, requestId, upstreamRequestId, readState)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	c.Header("X-Log-Source", readState.Source())
	common.ApiSuccess(c, pageInfo)
	return
}

func GetManagedUserLogs(c *gin.Context) {
	readState := model.NewLogReadState(c.Request.Context())
	pageInfo := common.GetPageQuery(c)
	userIds, ok := getManagedLogUserIds(c)
	if !ok {
		return
	}
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	logs, total, err := model.GetManagedUserLogs(
		userIds,
		logType,
		startTimestamp,
		endTimestamp,
		c.Query("model_name"),
		c.Query("token_name"),
		pageInfo.GetStartIdx(),
		pageInfo.GetPageSize(),
		c.Query("group"),
		c.Query("request_id"),
		c.Query("upstream_request_id"),
		readState,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	c.Header("X-Log-Source", readState.Source())
	common.ApiSuccess(c, pageInfo)
}

func getManagedLogUserIds(c *gin.Context) ([]int, bool) {
	managerUserId := c.GetInt("id")
	requestedUserIdText := c.Query("user_id")
	if requestedUserIdText != "" {
		requestedUserId, err := strconv.Atoi(requestedUserIdText)
		if err != nil || requestedUserId <= 0 {
			common.ApiError(c, errInvalidManagedUserId)
			return nil, false
		}
		allowed, err := model.CanAccessManagedCustomer(managerUserId, requestedUserId)
		if err != nil {
			common.ApiError(c, err)
			return nil, false
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权查看该客户数据"})
			return nil, false
		}
		return []int{requestedUserId}, true
	}
	userIds, err := model.GetManagedCustomerIds(managerUserId)
	if err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	return userIds, true
}

var errInvalidManagedUserId = &managedLogRequestError{message: "客户用户 ID 无效"}

type managedLogRequestError struct {
	message string
}

func (e *managedLogRequestError) Error() string {
	return e.message
}

// Deprecated: SearchAllLogs 已废弃，前端未使用该接口。
func SearchAllLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

// Deprecated: SearchUserLogs 已废弃，前端未使用该接口。
func SearchUserLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

func GetLogByKey(c *gin.Context) {
	readState := model.NewLogReadState(c.Request.Context())
	tokenId := c.GetInt("token_id")
	if tokenId == 0 {
		c.JSON(200, gin.H{
			"success": false,
			"message": "无效的令牌",
		})
		return
	}
	logs, err := model.GetLogByTokenId(tokenId, readState)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.Header("X-Log-Source", readState.Source())
	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data":    logs,
	})
}

func GetLogsStat(c *gin.Context) {
	readState := model.NewLogReadState(c.Request.Context())
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	username := c.Query("username")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	stat, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group, c.Query("stream_status"), readState)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	//tokenNum := model.SumUsedToken(logType, startTimestamp, endTimestamp, modelName, username, "")
	c.Header("X-Log-Source", readState.Source())
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"quota": stat.Quota,
			"rpm":   stat.Rpm,
			"tpm":   stat.Tpm,
		},
	})
	return
}

func GetLogsSelfStat(c *gin.Context) {
	readState := model.NewLogReadState(c.Request.Context())
	username := c.GetString("username")
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	quotaNum, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group, "", readState)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	//tokenNum := model.SumUsedToken(logType, startTimestamp, endTimestamp, modelName, username, tokenName)
	c.Header("X-Log-Source", readState.Source())
	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"quota": quotaNum.Quota,
			"rpm":   quotaNum.Rpm,
			"tpm":   quotaNum.Tpm,
			//"token": tokenNum,
		},
	})
	return
}

func GetManagedLogsStat(c *gin.Context) {
	readState := model.NewLogReadState(c.Request.Context())
	userIds, ok := getManagedLogUserIds(c)
	if !ok {
		return
	}
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	stat, err := model.SumUsedQuotaByUserIds(
		userIds,
		startTimestamp,
		endTimestamp,
		c.Query("model_name"),
		c.Query("token_name"),
		c.Query("group"),
		readState,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.Header("X-Log-Source", readState.Source())
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": stat})
}

func DeleteHistoryLogs(c *gin.Context) {
	targetTimestamp, _ := strconv.ParseInt(c.Query("target_timestamp"), 10, 64)
	if targetTimestamp == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "target timestamp is required",
		})
		return
	}
	count, err := model.DeleteOldLog(c.Request.Context(), targetTimestamp, 100)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    count,
	})
	return
}
