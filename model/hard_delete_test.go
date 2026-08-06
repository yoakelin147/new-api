package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHardDeleteUserRemovesAuthenticationRows(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(
		&TwoFA{}, &TwoFABackupCode{}, &PasskeyCredential{}, &UserOAuthBinding{}, &ManagedUserRelation{},
	))
	user := &User{Username: "hard-delete-auth", Password: "password"}
	require.NoError(t, DB.Create(user).Error)
	require.NoError(t, DB.Create(&Token{UserId: user.Id, Key: "hard-delete-key"}).Error)
	require.NoError(t, DB.Create(&TwoFA{UserId: user.Id, Secret: "secret"}).Error)
	require.NoError(t, DB.Create(&TwoFABackupCode{UserId: user.Id, CodeHash: "hash"}).Error)
	require.NoError(t, DB.Create(&PasskeyCredential{UserID: user.Id, CredentialID: "credential", PublicKey: "public"}).Error)
	require.NoError(t, DB.Create(&UserOAuthBinding{UserId: user.Id, ProviderId: 1, ProviderUserId: "provider-user"}).Error)
	require.NoError(t, DB.Create(&ManagedUserRelation{ManagerUserId: user.Id, CustomerUserId: user.Id + 1}).Error)

	require.NoError(t, HardDeleteUserById(user.Id))
	for _, record := range []any{&Token{}, &TwoFA{}, &TwoFABackupCode{}, &PasskeyCredential{}, &UserOAuthBinding{}} {
		var count int64
		require.NoError(t, DB.Unscoped().Model(record).Where("user_id = ?", user.Id).Count(&count).Error)
		require.Zero(t, count)
	}
	var relationCount int64
	require.NoError(t, DB.Unscoped().Model(&ManagedUserRelation{}).Where("manager_user_id = ? OR customer_user_id = ?", user.Id, user.Id).Count(&relationCount).Error)
	require.Zero(t, relationCount)
	var userCount int64
	require.NoError(t, DB.Unscoped().Model(&User{}).Where("id = ?", user.Id).Count(&userCount).Error)
	require.Zero(t, userCount)
}
