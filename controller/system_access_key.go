package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const systemAccessKeyPrefixLength = 12

type createMonitoringAccessKeyRequest struct {
	Name      string `json:"name"`
	ExpiresAt *int64 `json:"expires_at"`
}

type updateMonitoringAccessKeyStatusRequest struct {
	Enabled bool `json:"enabled"`
}

func ListMonitoringAccessKeys(c *gin.Context) {
	keys, err := model.ListSystemAccessKeys(constant.SystemAccessKeyTypeMonitoring)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    keys,
	})
}

func CreateMonitoringAccessKey(c *gin.Context) {
	var request createMonitoringAccessKeyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request"})
		return
	}

	name := strings.TrimSpace(request.Name)
	if name == "" || len(name) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "name must be between 1 and 128 characters"})
		return
	}
	now := common.GetTimestamp()
	if request.ExpiresAt != nil && *request.ExpiresAt <= now {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "expires_at must be in the future"})
		return
	}

	randomPart, err := common.GenerateRandomCharsKey(48)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	rawKey := "sak_" + randomPart
	key := &model.SystemAccessKey{
		Name:        name,
		KeyType:     constant.SystemAccessKeyTypeMonitoring,
		KeyHash:     model.HashSystemAccessKey(rawKey),
		KeyPrefix:   rawKey[:systemAccessKeyPrefixLength],
		Status:      constant.SystemAccessKeyStatusEnabled,
		ExpiresAt:   request.ExpiresAt,
		CreatedTime: now,
		UpdatedTime: now,
		CreatedBy:   c.GetInt("id"),
		UpdatedBy:   c.GetInt("id"),
	}
	if err = model.CreateSystemAccessKey(key); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"access_key": key,
			"key":        rawKey,
		},
	})
}

func UpdateMonitoringAccessKeyStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid id"})
		return
	}
	var request updateMonitoringAccessKeyStatusRequest
	if err = c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request"})
		return
	}

	status := constant.SystemAccessKeyStatusDisabled
	if request.Enabled {
		key, getErr := model.GetSystemAccessKeyById(id, constant.SystemAccessKeyTypeMonitoring)
		if errors.Is(getErr, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "access key not found"})
			return
		}
		if getErr != nil {
			common.ApiError(c, getErr)
			return
		}
		if key.ExpiresAt != nil && *key.ExpiresAt <= common.GetTimestamp() {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "expired access key cannot be enabled"})
			return
		}
		status = constant.SystemAccessKeyStatusEnabled
	}
	err = model.UpdateSystemAccessKeyStatus(
		id,
		constant.SystemAccessKeyTypeMonitoring,
		status,
		c.GetInt("id"),
		common.GetTimestamp(),
	)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "access key not found"})
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func DeleteMonitoringAccessKey(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid id"})
		return
	}
	err = model.DeleteSystemAccessKey(id, constant.SystemAccessKeyTypeMonitoring)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "access key not found"})
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
