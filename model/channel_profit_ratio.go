package model

import (
	"errors"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

type ChannelProfitRatio struct {
	Id            int     `json:"id"`
	ChannelId     int     `json:"channel_id" gorm:"not null;uniqueIndex:idx_channel_profit_ratio_scope"`
	ModelName     string  `json:"model_name" gorm:"type:varchar(255);default:'';uniqueIndex:idx_channel_profit_ratio_scope"`
	Group         string  `json:"group" gorm:"type:varchar(64);default:'';uniqueIndex:idx_channel_profit_ratio_scope"`
	UpstreamRatio float64 `json:"upstream_ratio" gorm:"not null;default:0"`
	Enabled       bool    `json:"enabled" gorm:"not null;default:true"`
	Remark        string  `json:"remark" gorm:"type:varchar(255);default:''"`
	CreatedAt     int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt     int64   `json:"updated_at" gorm:"bigint"`
	ChannelName   string  `json:"channel_name" gorm:"-"`
}

type ChannelProfitRatioInput struct {
	Id            int     `json:"id"`
	ChannelId     int     `json:"channel_id"`
	ModelName     string  `json:"model_name"`
	Group         string  `json:"group"`
	UpstreamRatio float64 `json:"upstream_ratio"`
	Enabled       bool    `json:"enabled"`
	Remark        string  `json:"remark"`
}

func ListChannelProfitRatios() ([]*ChannelProfitRatio, error) {
	var ratios []*ChannelProfitRatio
	if err := DB.Order("channel_id asc, model_name asc, " + commonGroupCol + " asc").Find(&ratios).Error; err != nil {
		common.SysError("failed to list channel profit ratios: " + err.Error())
		return nil, errors.New("查询渠道上游倍率失败")
	}
	channelIds := make(map[int]struct{})
	for _, ratio := range ratios {
		if ratio.ChannelId != 0 {
			channelIds[ratio.ChannelId] = struct{}{}
		}
	}
	names, err := getChannelNameMap(channelIds)
	if err != nil {
		return nil, err
	}
	for _, ratio := range ratios {
		ratio.ChannelName = names[ratio.ChannelId]
	}
	return ratios, nil
}

func SaveChannelProfitRatio(input ChannelProfitRatioInput) (*ChannelProfitRatio, error) {
	if input.ChannelId <= 0 {
		return nil, errors.New("渠道 ID 不能为空")
	}
	if input.UpstreamRatio < 0 {
		return nil, errors.New("上游倍率不能小于 0")
	}
	now := common.GetTimestamp()
	ratio := &ChannelProfitRatio{
		Id:            input.Id,
		ChannelId:     input.ChannelId,
		ModelName:     strings.TrimSpace(input.ModelName),
		Group:         strings.TrimSpace(input.Group),
		UpstreamRatio: input.UpstreamRatio,
		Enabled:       input.Enabled,
		Remark:        strings.TrimSpace(input.Remark),
		UpdatedAt:     now,
	}
	if ratio.Id == 0 {
		ratio.CreatedAt = now
		err := DB.Where("channel_id = ? AND model_name = ? AND "+commonGroupCol+" = ?", ratio.ChannelId, ratio.ModelName, ratio.Group).
			Assign(map[string]interface{}{
				"upstream_ratio": ratio.UpstreamRatio,
				"enabled":        ratio.Enabled,
				"remark":         ratio.Remark,
				"updated_at":     ratio.UpdatedAt,
			}).
			FirstOrCreate(ratio).Error
		if err != nil {
			common.SysError("failed to create channel profit ratio: " + err.Error())
			return nil, errors.New("保存渠道上游倍率失败")
		}
	} else {
		err := DB.Model(&ChannelProfitRatio{}).Where("id = ?", ratio.Id).Updates(map[string]interface{}{
			"channel_id":     ratio.ChannelId,
			"model_name":     ratio.ModelName,
			"group":          ratio.Group,
			"upstream_ratio": ratio.UpstreamRatio,
			"enabled":        ratio.Enabled,
			"remark":         ratio.Remark,
			"updated_at":     ratio.UpdatedAt,
		}).Error
		if err != nil {
			common.SysError("failed to update channel profit ratio: " + err.Error())
			return nil, errors.New("保存渠道上游倍率失败")
		}
		if err := DB.First(ratio, ratio.Id).Error; err != nil {
			return nil, errors.New("查询渠道上游倍率失败")
		}
	}
	names, err := getChannelNameMap(map[int]struct{}{ratio.ChannelId: struct{}{}})
	if err != nil {
		return nil, err
	}
	ratio.ChannelName = names[ratio.ChannelId]
	return ratio, nil
}

func DeleteChannelProfitRatio(id int) error {
	if id <= 0 {
		return errors.New("无效的渠道上游倍率配置")
	}
	if err := DB.Delete(&ChannelProfitRatio{}, id).Error; err != nil {
		common.SysError("failed to delete channel profit ratio: " + err.Error())
		return errors.New("删除渠道上游倍率失败")
	}
	return nil
}

func getEnabledChannelProfitRatios(channelIds map[int]struct{}) (map[int][]ChannelProfitRatio, error) {
	result := make(map[int][]ChannelProfitRatio)
	if len(channelIds) == 0 {
		return result, nil
	}
	ids := make([]int, 0, len(channelIds))
	for id := range channelIds {
		ids = append(ids, id)
	}
	var ratios []ChannelProfitRatio
	if err := DB.Where("enabled = ? AND channel_id IN ?", true, ids).Find(&ratios).Error; err != nil {
		common.SysError("failed to query channel profit ratios: " + err.Error())
		return nil, errors.New("查询渠道上游倍率失败")
	}
	for _, ratio := range ratios {
		result[ratio.ChannelId] = append(result[ratio.ChannelId], ratio)
	}
	for channelId := range result {
		sort.Slice(result[channelId], func(i, j int) bool {
			left := channelProfitRatioScore(result[channelId][i])
			right := channelProfitRatioScore(result[channelId][j])
			if left != right {
				return left > right
			}
			return result[channelId][i].Id < result[channelId][j].Id
		})
	}
	return result, nil
}

func channelProfitRatioScore(ratio ChannelProfitRatio) int {
	score := 0
	if ratio.ModelName != "" {
		score += 2
	}
	if ratio.Group != "" {
		score++
	}
	return score
}

func matchChannelProfitRatio(ratios []ChannelProfitRatio, modelName string, group string) (*ChannelProfitRatio, bool) {
	for i := range ratios {
		ratio := &ratios[i]
		if ratio.ModelName != "" && ratio.ModelName != modelName {
			continue
		}
		if ratio.Group != "" && ratio.Group != group {
			continue
		}
		return ratio, true
	}
	return nil, false
}
