package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func withUserUpdateTestDatabase(t *testing.T) {
	t.Helper()
	previousDB := DB
	dsn := fmt.Sprintf("file:user-update-%d?mode=memory&cache=shared", time.Now().UnixNano())
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB = testDB
	require.NoError(t, DB.AutoMigrate(&User{}))
	t.Cleanup(func() { DB = previousDB })
}

func TestUpdateUserAccessTokenDoesNotOverwriteConcurrentQuota(t *testing.T) {
	withUserUpdateTestDatabase(t)

	user := User{
		Id:       1,
		Username: "token-rotation-user",
		Password: "password",
		Status:   common.UserStatusEnabled,
		Quota:    1000,
		AffQuota: 800,
	}
	require.NoError(t, DB.Create(&user).Error)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
		"quota":     gorm.Expr("quota + ?", 500),
		"aff_quota": gorm.Expr("aff_quota - ?", 500),
	}).Error)

	require.NoError(t, UpdateUserAccessToken(user.Id, "rotated-token"))

	var got User
	require.NoError(t, DB.First(&got, user.Id).Error)
	require.Equal(t, "rotated-token", got.GetAccessToken())
	require.Equal(t, 1500, got.Quota)
	require.Equal(t, 300, got.AffQuota)
}

func TestUpdateOmitsAccountingAndAccessTokenFields(t *testing.T) {
	withUserUpdateTestDatabase(t)

	user := User{
		Id:       2,
		Username: "stale-user",
		Password: "password",
		Status:   common.UserStatusEnabled,
		Quota:    1000,
		AffQuota: 800,
	}
	user.SetAccessToken("old-token")
	require.NoError(t, DB.Create(&user).Error)

	stale, err := GetUserById(user.Id, true)
	require.NoError(t, err)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
		"quota":        gorm.Expr("quota + ?", 500),
		"aff_quota":    gorm.Expr("aff_quota - ?", 500),
		"access_token": "rotated-token",
	}).Error)
	stale.DisplayName = "updated"

	require.NoError(t, stale.Update(false))

	var got User
	require.NoError(t, DB.First(&got, user.Id).Error)
	require.Equal(t, "rotated-token", got.GetAccessToken())
	require.Equal(t, 1500, got.Quota)
	require.Equal(t, 300, got.AffQuota)
	require.Equal(t, "updated", got.DisplayName)
}
