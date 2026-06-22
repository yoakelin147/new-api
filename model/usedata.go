package model

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// QuotaData 柱状图数据
type QuotaData struct {
	Id        int    `json:"id"`
	UserID    int    `json:"user_id" gorm:"index"`
	Username  string `json:"username" gorm:"index:idx_qdt_model_user_name,priority:2;size:64;default:''"`
	ModelName string `json:"model_name" gorm:"index:idx_qdt_model_user_name,priority:1;size:64;default:''"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;index:idx_qdt_created_at,priority:2"`
	TokenUsed int    `json:"token_used" gorm:"default:0"`
	Count     int    `json:"count" gorm:"default:0"`
	Quota     int    `json:"quota" gorm:"default:0"`
}

type ChannelQuotaData struct {
	ChannelId   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	CreatedAt   int64  `json:"created_at"`
	TokenUsed   int    `json:"token_used"`
	Count       int    `json:"count"`
	Quota       int    `json:"quota"`
}

type TokenQuotaData struct {
	TokenId   int    `json:"token_id"`
	TokenName string `json:"token_name"`
	CreatedAt int64  `json:"created_at"`
	TokenUsed int    `json:"token_used"`
	Count     int    `json:"count"`
	Quota     int    `json:"quota"`
}

type UserTokenQuotaData struct {
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	TokenId   int    `json:"token_id"`
	TokenName string `json:"token_name"`
	CreatedAt int64  `json:"created_at"`
	TokenUsed int    `json:"token_used"`
	Count     int    `json:"count"`
	Quota     int    `json:"quota"`
}

func UpdateQuotaData() {
	for {
		if common.DataExportEnabled {
			common.SysLog("正在更新数据看板数据...")
			SaveQuotaDataCache()
		}
		time.Sleep(time.Duration(common.DataExportInterval) * time.Minute)
	}
}

var CacheQuotaData = make(map[string]*QuotaData)
var CacheQuotaDataLock = sync.Mutex{}

func logQuotaDataCache(userId int, username string, modelName string, quota int, createdAt int64, tokenUsed int) {
	key := fmt.Sprintf("%d-%s-%s-%d", userId, username, modelName, createdAt)
	quotaData, ok := CacheQuotaData[key]
	if ok {
		quotaData.Count += 1
		quotaData.Quota += quota
		quotaData.TokenUsed += tokenUsed
	} else {
		quotaData = &QuotaData{
			UserID:    userId,
			Username:  username,
			ModelName: modelName,
			CreatedAt: createdAt,
			Count:     1,
			Quota:     quota,
			TokenUsed: tokenUsed,
		}
	}
	CacheQuotaData[key] = quotaData
}

func LogQuotaData(userId int, username string, modelName string, quota int, createdAt int64, tokenUsed int) {
	// 只精确到小时
	createdAt = createdAt - (createdAt % 3600)

	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	logQuotaDataCache(userId, username, modelName, quota, createdAt, tokenUsed)
}

func SaveQuotaDataCache() {
	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	size := len(CacheQuotaData)
	// 如果缓存中有数据，就保存到数据库中
	// 1. 先查询数据库中是否有数据
	// 2. 如果有数据，就更新数据
	// 3. 如果没有数据，就插入数据
	for _, quotaData := range CacheQuotaData {
		quotaDataDB := &QuotaData{}
		DB.Table("quota_data").Where("user_id = ? and username = ? and model_name = ? and created_at = ?",
			quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.CreatedAt).First(quotaDataDB)
		if quotaDataDB.Id > 0 {
			//quotaDataDB.Count += quotaData.Count
			//quotaDataDB.Quota += quotaData.Quota
			//DB.Table("quota_data").Save(quotaDataDB)
			increaseQuotaData(quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.Count, quotaData.Quota, quotaData.CreatedAt, quotaData.TokenUsed)
		} else {
			DB.Table("quota_data").Create(quotaData)
		}
	}
	CacheQuotaData = make(map[string]*QuotaData)
	common.SysLog(fmt.Sprintf("保存数据看板数据成功，共保存%d条数据", size))
}

func increaseQuotaData(userId int, username string, modelName string, count int, quota int, createdAt int64, tokenUsed int) {
	err := DB.Table("quota_data").Where("user_id = ? and username = ? and model_name = ? and created_at = ?",
		userId, username, modelName, createdAt).Updates(map[string]interface{}{
		"count":      gorm.Expr("count + ?", count),
		"quota":      gorm.Expr("quota + ?", quota),
		"token_used": gorm.Expr("token_used + ?", tokenUsed),
	}).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("increaseQuotaData error: %s", err))
	}
}

func GetQuotaDataByUsername(username string, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	// 从quota_data表中查询数据
	err = DB.Table("quota_data").Where("username = ? and created_at >= ? and created_at <= ?", username, startTime, endTime).Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetQuotaDataByUserId(userId int, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	// 从quota_data表中查询数据
	err = DB.Table("quota_data").Where("user_id = ? and created_at >= ? and created_at <= ?", userId, startTime, endTime).Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetQuotaDataGroupByUser(startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	err = DB.Table("quota_data").
		Select("username, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used").
		Where("created_at >= ? and created_at <= ?", startTime, endTime).
		Group("username, created_at").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetQuotaDataGroupByChannel(startTime int64, endTime int64) ([]*ChannelQuotaData, error) {
	var quotaDatas []*ChannelQuotaData
	bucketExpr := quotaDataBucketExpr()
	query := LOG_DB.Table("logs").
		Select(fmt.Sprintf("channel_id, %s as created_at, count(*) as count, sum(quota) as quota, sum(prompt_tokens) + sum(completion_tokens) as token_used", bucketExpr)).
		Where("type = ? and channel_id <> 0", LogTypeConsume).
		Group(fmt.Sprintf("channel_id, %s", bucketExpr)).
		Order("created_at asc")
	if startTime > 0 {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("created_at <= ?", endTime)
	}
	if err := query.Find(&quotaDatas).Error; err != nil {
		return nil, err
	}

	channelIds := make([]int, 0)
	channelSeen := make(map[int]struct{})
	for _, quotaData := range quotaDatas {
		if quotaData.ChannelId == 0 {
			continue
		}
		if _, ok := channelSeen[quotaData.ChannelId]; ok {
			continue
		}
		channelSeen[quotaData.ChannelId] = struct{}{}
		channelIds = append(channelIds, quotaData.ChannelId)
	}
	if len(channelIds) == 0 {
		return quotaDatas, nil
	}

	channelNames := make(map[int]string, len(channelIds))
	if common.MemoryCacheEnabled {
		missingChannelIds := make([]int, 0)
		for _, channelId := range channelIds {
			cacheChannel, err := CacheGetChannel(channelId)
			if err == nil && cacheChannel != nil {
				channelNames[channelId] = cacheChannel.Name
				continue
			}
			missingChannelIds = append(missingChannelIds, channelId)
		}
		channelIds = missingChannelIds
	}
	if len(channelIds) > 0 {
		var channels []struct {
			Id   int    `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if err := DB.Table("channels").Select("id, name").Where("id IN ?", channelIds).Find(&channels).Error; err != nil {
			return quotaDatas, err
		}
		for _, channel := range channels {
			channelNames[channel.Id] = channel.Name
		}
	}
	for _, quotaData := range quotaDatas {
		quotaData.ChannelName = channelNames[quotaData.ChannelId]
	}

	return quotaDatas, nil
}

func GetQuotaDataGroupByToken(userId int, startTime int64, endTime int64) ([]*TokenQuotaData, error) {
	var quotaDatas []*TokenQuotaData
	bucketExpr := quotaDataBucketExpr()
	query := LOG_DB.Table("logs").
		Select(fmt.Sprintf("token_id, token_name, %s as created_at, count(*) as count, sum(quota) as quota, sum(prompt_tokens) + sum(completion_tokens) as token_used", bucketExpr)).
		Where("type = ? and user_id = ? and (token_id <> 0 or token_name <> '')", LogTypeConsume, userId).
		Group(fmt.Sprintf("token_id, token_name, %s", bucketExpr)).
		Order("created_at asc, token_id asc, token_name asc")
	if startTime > 0 {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("created_at <= ?", endTime)
	}
	if err := query.Find(&quotaDatas).Error; err != nil {
		return nil, err
	}
	return quotaDatas, nil
}

func GetQuotaDataGroupByUserToken(userId int, startTime int64, endTime int64) ([]*UserTokenQuotaData, error) {
	var quotaDatas []*UserTokenQuotaData
	bucketExpr := quotaDataBucketExpr()
	query := LOG_DB.Table("logs").
		Select(fmt.Sprintf("user_id, username, token_id, token_name, %s as created_at, count(*) as count, sum(quota) as quota, sum(prompt_tokens) + sum(completion_tokens) as token_used", bucketExpr)).
		Where("type = ? and (token_id <> 0 or token_name <> '')", LogTypeConsume).
		Group(fmt.Sprintf("user_id, username, token_id, token_name, %s", bucketExpr)).
		Order("created_at asc, user_id asc, token_id asc, token_name asc")
	if userId > 0 {
		query = query.Where("user_id = ?", userId)
	} else {
		query = query.Where("user_id <> 0")
	}
	if startTime > 0 {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("created_at <= ?", endTime)
	}
	if err := query.Find(&quotaDatas).Error; err != nil {
		return nil, err
	}
	return quotaDatas, nil
}

func quotaDataBucketExpr() string {
	if common.UsingMySQL {
		return "FLOOR(created_at / 3600) * 3600"
	}
	return "(created_at / 3600) * 3600"
}

func GetAllQuotaDates(startTime int64, endTime int64, username string) (quotaData []*QuotaData, err error) {
	if username != "" {
		return GetQuotaDataByUsername(username, startTime, endTime)
	}
	var quotaDatas []*QuotaData
	// 从quota_data表中查询数据
	// only select model_name, sum(count) as count, sum(quota) as quota, model_name, created_at from quota_data group by model_name, created_at;
	//err = DB.Table("quota_data").Where("created_at >= ? and created_at <= ?", startTime, endTime).Find(&quotaDatas).Error
	err = DB.Table("quota_data").Select("model_name, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used, created_at").Where("created_at >= ? and created_at <= ?", startTime, endTime).Group("model_name, created_at").Find(&quotaDatas).Error
	return quotaDatas, err
}
