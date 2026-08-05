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

func withManagedUserTestDatabases(t *testing.T) {
	t.Helper()
	previousDB := DB
	previousLogDB := LOG_DB
	previousLogGroupCol := logGroupCol
	dsn := fmt.Sprintf("file:managed-users-%d?mode=memory&cache=shared", time.Now().UnixNano())
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB = testDB
	LOG_DB = testDB
	logGroupCol = "`group`"
	require.NoError(t, DB.AutoMigrate(&User{}, &ManagedUserRelation{}, &Log{}))
	t.Cleanup(func() {
		DB = previousDB
		LOG_DB = previousLogDB
		logGroupCol = previousLogGroupCol
	})
}

func TestManagedUserRelationLifecycleAllowsSharedCustomer(t *testing.T) {
	withManagedUserTestDatabases(t)

	users := []User{
		{Id: 1, Username: "sales-a", Password: "hash", AffCode: "a001", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 2, Username: "sales-b", Password: "hash", AffCode: "a002", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 10, Username: "customer-a", Password: "hash", AffCode: "c010", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
		{Id: 11, Username: "customer-b", Password: "hash", AffCode: "c011", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
		{Id: 20, Username: "admin", Password: "hash", AffCode: "a020", Role: common.RoleAdminUser, Status: common.UserStatusEnabled},
	}
	require.NoError(t, DB.Create(&users).Error)

	require.NoError(t, AssignManagedCustomers(1, []int{10}, 20))
	salesUsers, salesTotal, err := ListSalesUsers("", 0, 100)
	require.NoError(t, err)
	require.Equal(t, int64(2), salesTotal)
	require.Len(t, salesUsers, 2)
	require.Equal(t, int64(1), salesUsers[1].CustomerCount)
	allowed, err := CanAccessManagedCustomer(1, 10)
	require.NoError(t, err)
	require.True(t, allowed)
	allowed, err = CanAccessManagedCustomer(2, 10)
	require.NoError(t, err)
	require.False(t, allowed)

	customers, err := ListManagedCustomers(1)
	require.NoError(t, err)
	require.Equal(t, []ManagedCustomerSummary{{
		Id: 10, Username: "customer-a", DisplayName: "", Status: common.UserStatusEnabled,
	}}, customers)

	available, total, err := SearchAvailableManagedCustomers(1, "", 0, 100)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, []ManagedCustomerSummary{{
		Id: 11, Username: "customer-b", DisplayName: "", Status: common.UserStatusEnabled,
	}}, available)

	available, total, err = SearchAvailableManagedCustomers(2, "", 0, 100)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Equal(t, []ManagedCustomerSummary{
		{Id: 11, Username: "customer-b", DisplayName: "", Status: common.UserStatusEnabled},
		{Id: 10, Username: "customer-a", DisplayName: "", Status: common.UserStatusEnabled},
	}, available)

	require.NoError(t, AssignManagedCustomers(2, []int{10}, 20))
	require.Error(t, AssignManagedCustomers(1, []int{10}, 20))
	allowed, err = CanAccessManagedCustomer(2, 10)
	require.NoError(t, err)
	require.True(t, allowed)
	require.Error(t, AssignManagedCustomers(1, []int{20}, 20))
	require.NoError(t, RemoveManagedCustomer(1, 10))
	allowed, err = CanAccessManagedCustomer(1, 10)
	require.NoError(t, err)
	require.False(t, allowed)
	allowed, err = CanAccessManagedCustomer(2, 10)
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestMigrateManagedUserRelationIndexesAllowsSharedCustomer(t *testing.T) {
	previousDB := DB
	dsn := fmt.Sprintf("file:managed-users-migration-%d?mode=memory&cache=shared", time.Now().UnixNano())
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB = testDB
	t.Cleanup(func() { DB = previousDB })

	type legacyManagedUserRelation struct {
		Id             int   `gorm:"primaryKey"`
		ManagerUserId  int   `gorm:"not null;index;uniqueIndex:idx_manager_customer"`
		CustomerUserId int   `gorm:"not null;uniqueIndex;uniqueIndex:idx_manager_customer"`
		CreatedBy      int   `gorm:"not null;default:0"`
		CreatedAt      int64 `gorm:"autoCreateTime"`
		UpdatedAt      int64 `gorm:"autoUpdateTime"`
	}
	require.NoError(t, DB.Table("managed_user_relations").AutoMigrate(&legacyManagedUserRelation{}))
	require.NoError(t, migrateManagedUserRelationIndexes())
	require.NoError(t, migrateManagedUserRelationIndexes())

	require.NoError(t, DB.Create(&ManagedUserRelation{ManagerUserId: 1, CustomerUserId: 10}).Error)
	require.NoError(t, DB.Create(&ManagedUserRelation{ManagerUserId: 2, CustomerUserId: 10}).Error)
	require.Error(t, DB.Create(&ManagedUserRelation{ManagerUserId: 1, CustomerUserId: 10}).Error)

	indexes, err := listManagedUserRelationIndexes()
	require.NoError(t, err)
	for _, index := range indexes {
		require.False(t, index.unique && len(index.columns) == 1 && index.columns[0] == "customer_user_id")
	}
}

func TestListManagedCustomersPage(t *testing.T) {
	withManagedUserTestDatabases(t)

	users := []User{
		{Id: 1, Username: "sales", Password: "hash", AffCode: "s001", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 10, Username: "customer-a", Password: "hash", AffCode: "c010", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
		{Id: 11, Username: "customer-b", Password: "hash", AffCode: "c011", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
		{Id: 12, Username: "customer-c", Password: "hash", AffCode: "c012", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
	}
	require.NoError(t, DB.Create(&users).Error)
	require.NoError(t, AssignManagedCustomers(1, []int{10, 11, 12}, 99))

	firstPage, total, err := ListManagedCustomersPage(1, 0, 2)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Equal(t, []ManagedCustomerSummary{
		{Id: 12, Username: "customer-c", Status: common.UserStatusEnabled},
		{Id: 11, Username: "customer-b", Status: common.UserStatusEnabled},
	}, firstPage)

	secondPage, total, err := ListManagedCustomersPage(1, 2, 2)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Equal(t, []ManagedCustomerSummary{
		{Id: 10, Username: "customer-a", Status: common.UserStatusEnabled},
	}, secondPage)
}

func TestRemoveManagedCustomersIsAtomic(t *testing.T) {
	withManagedUserTestDatabases(t)

	users := []User{
		{Id: 1, Username: "sales", Password: "hash", AffCode: "s001", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 10, Username: "customer-a", Password: "hash", AffCode: "c010", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
		{Id: 11, Username: "customer-b", Password: "hash", AffCode: "c011", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
	}
	require.NoError(t, DB.Create(&users).Error)
	require.NoError(t, AssignManagedCustomers(1, []int{10, 11}, 99))

	require.NoError(t, RemoveManagedCustomers(1, []int{10, 11}))
	var relationCount int64
	require.NoError(t, DB.Model(&ManagedUserRelation{}).Count(&relationCount).Error)
	require.Zero(t, relationCount)

	require.NoError(t, AssignManagedCustomers(1, []int{10, 11}, 99))
	require.Error(t, RemoveManagedCustomers(1, []int{10, 999}))
	require.NoError(t, DB.Model(&ManagedUserRelation{}).Count(&relationCount).Error)
	require.Equal(t, int64(2), relationCount)
}

func TestManagedUserLogsAreScopedAndRedacted(t *testing.T) {
	withManagedUserTestDatabases(t)

	now := time.Now().Unix()
	logs := []Log{
		{
			UserId: 10, Username: "customer-a", Type: LogTypeConsume, CreatedAt: now,
			Quota: 100, PromptTokens: 10, CompletionTokens: 20, ChannelId: 88,
			ChannelName: "private-channel", TokenId: 123, TokenName: "customer-key",
			Ip: "192.0.2.1", Group: "default", Other: common.MapToJsonStr(map[string]interface{}{
				"model_ratio":         1,
				"admin_info":          map[string]interface{}{"secret": true},
				"audit_info":          map[string]interface{}{"operator": "root"},
				"upstream_model_name": "private-upstream",
			}),
		},
		{UserId: 11, Username: "customer-b", Type: LogTypeConsume, CreatedAt: now, Quota: 900, Group: "default"},
	}
	require.NoError(t, LOG_DB.Create(&logs).Error)

	result, total, err := GetManagedUserLogs([]int{10}, LogTypeUnknown, 0, 0, "", "", 0, 20, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, result, 1)
	require.Equal(t, 10, result[0].UserId)
	require.Zero(t, result[0].ChannelId)
	require.Empty(t, result[0].ChannelName)
	require.Zero(t, result[0].TokenId)
	require.Empty(t, result[0].Ip)
	other, err := common.StrToMap(result[0].Other)
	require.NoError(t, err)
	require.NotContains(t, other, "admin_info")
	require.NotContains(t, other, "audit_info")
	require.NotContains(t, other, "upstream_model_name")
	require.Contains(t, other, "model_ratio")

	stat, err := SumUsedQuotaByUserIds([]int{10}, 0, 0, "", "", "")
	require.NoError(t, err)
	require.Equal(t, 100, stat.Quota)
	require.Equal(t, 1, stat.Rpm)
	require.Equal(t, 30, stat.Tpm)
}

func TestEditingCustomerRoleClearsManagedRelationsAtomically(t *testing.T) {
	withManagedUserTestDatabases(t)

	users := []User{
		{Id: 1, Username: "sales", Password: "hash", AffCode: "s001", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 10, Username: "customer", Password: "hash", AffCode: "c010", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
	}
	require.NoError(t, DB.Create(&users).Error)
	require.NoError(t, AssignManagedCustomers(1, []int{10}, 99))

	updatedCustomer := users[1]
	updatedCustomer.Role = common.RoleSalesUser
	require.NoError(t, updatedCustomer.Edit(false))

	var savedCustomer User
	require.NoError(t, DB.First(&savedCustomer, 10).Error)
	require.Equal(t, common.RoleSalesUser, savedCustomer.Role)
	var relationCount int64
	require.NoError(t, DB.Model(&ManagedUserRelation{}).Count(&relationCount).Error)
	require.Zero(t, relationCount)
}

func TestManagedRoleActionClearsSalesCustomers(t *testing.T) {
	withManagedUserTestDatabases(t)

	sales := User{Id: 1, Username: "sales", Password: "hash", AffCode: "s001", Role: common.RoleSalesUser, Status: common.UserStatusEnabled}
	customer := User{Id: 10, Username: "customer", Password: "hash", AffCode: "c010", Role: common.RoleCommonUser, Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&[]User{sales, customer}).Error)
	require.NoError(t, AssignManagedCustomers(sales.Id, []int{customer.Id}, 99))

	require.NoError(t, sales.UpdateRole(common.RoleCommonUser))
	require.Equal(t, common.RoleCommonUser, sales.Role)
	var relationCount int64
	require.NoError(t, DB.Model(&ManagedUserRelation{}).Count(&relationCount).Error)
	require.Zero(t, relationCount)
}
