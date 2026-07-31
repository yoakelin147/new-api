package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

type SystemAccessKey struct {
	Id           int            `json:"id" gorm:"primaryKey"`
	Name         string         `json:"name" gorm:"type:varchar(128);index;not null"`
	KeyType      string         `json:"key_type" gorm:"type:varchar(32);index;not null"`
	KeyHash      string         `json:"-" gorm:"type:char(64);uniqueIndex;not null"`
	KeyPrefix    string         `json:"key_prefix" gorm:"type:varchar(16);not null"`
	Status       int            `json:"status" gorm:"default:1;index"`
	ExpiresAt    *int64         `json:"expires_at" gorm:"index"`
	CreatedTime  int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime  int64          `json:"updated_time" gorm:"bigint"`
	LastUsedTime int64          `json:"last_used_time" gorm:"bigint;default:0"`
	CreatedBy    int            `json:"created_by"`
	UpdatedBy    int            `json:"updated_by"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func HashSystemAccessKey(rawKey string) string {
	digest := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(digest[:])
}

func (key *SystemAccessKey) IsUsable(now int64) bool {
	if key == nil || key.Status != constant.SystemAccessKeyStatusEnabled {
		return false
	}
	return key.ExpiresAt == nil || *key.ExpiresAt > now
}

func ListSystemAccessKeys(keyType string) ([]SystemAccessKey, error) {
	keys := make([]SystemAccessKey, 0)
	err := DB.Where("key_type = ?", keyType).Order("id DESC").Find(&keys).Error
	return keys, err
}

func GetSystemAccessKeyByHash(keyType string, keyHash string) (*SystemAccessKey, error) {
	var key SystemAccessKey
	err := DB.Where("key_type = ? AND key_hash = ?", keyType, keyHash).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func GetSystemAccessKeyById(id int, keyType string) (*SystemAccessKey, error) {
	var key SystemAccessKey
	err := DB.Where("id = ? AND key_type = ?", id, keyType).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func CreateSystemAccessKey(key *SystemAccessKey) error {
	key.Name = strings.TrimSpace(key.Name)
	return DB.Create(key).Error
}

func UpdateSystemAccessKeyStatus(id int, keyType string, status int, updatedBy int, updatedTime int64) error {
	result := DB.Model(&SystemAccessKey{}).
		Where("id = ? AND key_type = ?", id, keyType).
		Updates(map[string]any{
			"status":       status,
			"updated_by":   updatedBy,
			"updated_time": updatedTime,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func DeleteSystemAccessKey(id int, keyType string) error {
	result := DB.Where("id = ? AND key_type = ?", id, keyType).Delete(&SystemAccessKey{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func TouchSystemAccessKey(id int, usedTime int64) error {
	return DB.Model(&SystemAccessKey{}).
		Where("id = ?", id).
		Update("last_used_time", usedTime).Error
}
