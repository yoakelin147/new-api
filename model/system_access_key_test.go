package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSystemAccessKeyLifecycle(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&SystemAccessKey{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&SystemAccessKey{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&SystemAccessKey{}).Error)
	})

	expiresAt := int64(2000)
	rawKey := "sak_test-system-access-key"
	key := &SystemAccessKey{
		Name:        "zabbix",
		KeyType:     constant.SystemAccessKeyTypeMonitoring,
		KeyHash:     HashSystemAccessKey(rawKey),
		KeyPrefix:   "sak_test",
		Status:      constant.SystemAccessKeyStatusEnabled,
		ExpiresAt:   &expiresAt,
		CreatedTime: 1000,
		UpdatedTime: 1000,
		CreatedBy:   1,
		UpdatedBy:   1,
	}
	require.NoError(t, CreateSystemAccessKey(key))

	stored, err := GetSystemAccessKeyByHash(constant.SystemAccessKeyTypeMonitoring, HashSystemAccessKey(rawKey))
	require.NoError(t, err)
	require.True(t, stored.IsUsable(1500))
	require.False(t, stored.IsUsable(2000))

	require.NoError(t, UpdateSystemAccessKeyStatus(
		stored.Id,
		constant.SystemAccessKeyTypeMonitoring,
		constant.SystemAccessKeyStatusDisabled,
		2,
		1600,
	))
	stored, err = GetSystemAccessKeyByHash(constant.SystemAccessKeyTypeMonitoring, HashSystemAccessKey(rawKey))
	require.NoError(t, err)
	require.False(t, stored.IsUsable(1600))

	require.NoError(t, TouchSystemAccessKey(stored.Id, 1700))
	keys, err := ListSystemAccessKeys(constant.SystemAccessKeyTypeMonitoring)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Equal(t, int64(1700), keys[0].LastUsedTime)

	require.NoError(t, DeleteSystemAccessKey(stored.Id, constant.SystemAccessKeyTypeMonitoring))
	_, err = GetSystemAccessKeyByHash(constant.SystemAccessKeyTypeMonitoring, HashSystemAccessKey(rawKey))
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestSystemAccessKeyPermanentExpiry(t *testing.T) {
	key := &SystemAccessKey{Status: constant.SystemAccessKeyStatusEnabled}
	require.True(t, key.IsUsable(1000))
}
