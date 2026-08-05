package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func withManagedLogControllerDatabase(t *testing.T) {
	t.Helper()
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	previousRedisEnabled := common.RedisEnabled
	dsn := fmt.Sprintf("file:managed-log-controller-%d?mode=memory&cache=shared", time.Now().UnixNano())
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = testDB
	model.LOG_DB = testDB
	common.RedisEnabled = false
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.ManagedUserRelation{}, &model.Log{}, &model.QuotaData{}))
	t.Cleanup(func() {
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		common.RedisEnabled = previousRedisEnabled
	})
}

func performManagedLogControllerRequest(t *testing.T, managerUserId int, targetUserId int) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/log/managed?user_id=%d&p=1&page_size=20", targetUserId),
		nil,
	)
	context.Request = request
	context.Set("id", managerUserId)
	GetManagedUserLogs(context)
	return recorder
}

func performManagedCostControllerRequest(t *testing.T, managerUserId int, targetUserId int) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/cost_statistics/managed?user_id=%d&period=today&p=1&page_size=20", targetUserId),
		nil,
	)
	context.Request = request
	context.Set("id", managerUserId)
	GetManagedCostStatistics(context)
	return recorder
}

func TestManagedLogsRejectUnassignedCustomer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withManagedLogControllerDatabase(t)

	users := []model.User{
		{Id: 1, Username: "sales-a", Password: "hash", AffCode: "s001", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 2, Username: "sales-b", Password: "hash", AffCode: "s002", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 10, Username: "customer-a", Password: "hash", AffCode: "c010", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
		{Id: 11, Username: "customer-b", Password: "hash", AffCode: "c011", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
	}
	require.NoError(t, model.DB.Create(&users).Error)
	require.NoError(t, model.AssignManagedCustomers(1, []int{10}, 99))
	require.NoError(t, model.AssignManagedCustomers(2, []int{10, 11}, 99))
	require.NoError(t, model.LOG_DB.Create(&model.Log{
		UserId: 11, Username: "customer-b", Type: model.LogTypeConsume, CreatedAt: time.Now().Unix(),
	}).Error)

	forbidden := performManagedLogControllerRequest(t, 1, 11)
	require.Equal(t, http.StatusForbidden, forbidden.Code)
	require.Contains(t, forbidden.Body.String(), "无权查看该客户数据")

	allowed := performManagedLogControllerRequest(t, 1, 10)
	require.Equal(t, http.StatusOK, allowed.Code)
	require.Contains(t, allowed.Body.String(), `"success":true`)
	require.NotContains(t, allowed.Body.String(), "customer-b")

	sharedCustomer := performManagedLogControllerRequest(t, 2, 10)
	require.Equal(t, http.StatusOK, sharedCustomer.Code)
	require.Contains(t, sharedCustomer.Body.String(), `"success":true`)
}

func TestManagedCostStatisticsRejectUnassignedCustomer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withManagedLogControllerDatabase(t)

	now := time.Now().Unix()
	users := []model.User{
		{Id: 1, Username: "sales-a", Password: "hash", AffCode: "s001", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 2, Username: "sales-b", Password: "hash", AffCode: "s002", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 10, Username: "customer-a", Password: "hash", AffCode: "c010", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
		{Id: 11, Username: "customer-b", Password: "hash", AffCode: "c011", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
	}
	require.NoError(t, model.DB.Create(&users).Error)
	require.NoError(t, model.AssignManagedCustomers(1, []int{10}, 99))
	require.NoError(t, model.AssignManagedCustomers(2, []int{11}, 99))
	require.NoError(t, model.LOG_DB.Create(&[]model.Log{
		{UserId: 10, Username: "customer-a", Type: model.LogTypeConsume, CreatedAt: now, TokenId: 10, TokenName: "a", ModelName: "gpt-a", Quota: 100},
		{UserId: 11, Username: "customer-b", Type: model.LogTypeConsume, CreatedAt: now, TokenId: 11, TokenName: "b", ModelName: "gpt-b", Quota: 900},
	}).Error)

	forbidden := performManagedCostControllerRequest(t, 1, 11)
	require.Equal(t, http.StatusForbidden, forbidden.Code)
	require.Contains(t, forbidden.Body.String(), "无权查看该客户数据")

	allowed := performManagedCostControllerRequest(t, 1, 10)
	require.Equal(t, http.StatusOK, allowed.Code)
	require.Contains(t, allowed.Body.String(), `"username":"customer-a"`)
	require.NotContains(t, allowed.Body.String(), "customer-b")
}

func performManagedQuotaControllerRequest(t *testing.T, managerUserId int, targetUserId int) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	path := "/api/data/managed?start_timestamp=3600&end_timestamp=7200"
	if targetUserId > 0 {
		path += fmt.Sprintf("&user_id=%d", targetUserId)
	}
	context.Request = httptest.NewRequest(http.MethodGet, path, nil)
	context.Set("id", managerUserId)
	GetManagedQuotaDates(context)
	return recorder
}

func TestManagedQuotaDatesAreScopedToAssignedCustomers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withManagedLogControllerDatabase(t)

	users := []model.User{
		{Id: 1, Username: "sales-a", Password: "hash", AffCode: "s001", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 2, Username: "sales-b", Password: "hash", AffCode: "s002", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 10, Username: "customer-a", Password: "hash", AffCode: "c010", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
		{Id: 11, Username: "customer-b", Password: "hash", AffCode: "c011", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
		{Id: 12, Username: "customer-c", Password: "hash", AffCode: "c012", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
	}
	require.NoError(t, model.DB.Create(&users).Error)
	require.NoError(t, model.AssignManagedCustomers(1, []int{10, 12}, 99))
	require.NoError(t, model.AssignManagedCustomers(2, []int{11}, 99))
	require.NoError(t, model.DB.Create(&[]model.QuotaData{
		{UserID: 10, Username: "customer-a", ModelName: "gpt-managed", CreatedAt: 3600, Count: 1, Quota: 100, TokenUsed: 10},
		{UserID: 12, Username: "customer-c", ModelName: "gpt-managed", CreatedAt: 3600, Count: 2, Quota: 200, TokenUsed: 20},
		{UserID: 11, Username: "customer-b", ModelName: "gpt-unassigned", CreatedAt: 3600, Count: 9, Quota: 900, TokenUsed: 90},
	}).Error)

	allAssigned := performManagedQuotaControllerRequest(t, 1, 0)
	require.Equal(t, http.StatusOK, allAssigned.Code)
	require.Contains(t, allAssigned.Body.String(), `"model_name":"gpt-managed"`)
	require.Contains(t, allAssigned.Body.String(), `"count":3`)
	require.NotContains(t, allAssigned.Body.String(), "gpt-unassigned")

	selectedCustomer := performManagedQuotaControllerRequest(t, 1, 10)
	require.Equal(t, http.StatusOK, selectedCustomer.Code)
	require.Contains(t, selectedCustomer.Body.String(), `"count":1`)
	require.NotContains(t, selectedCustomer.Body.String(), `"count":2`)

	forbidden := performManagedQuotaControllerRequest(t, 1, 11)
	require.Equal(t, http.StatusForbidden, forbidden.Code)
	require.Contains(t, forbidden.Body.String(), "无权查看该客户数据")
}

func TestGetSalesCustomersPaginatesAssignedCustomers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withManagedLogControllerDatabase(t)

	users := []model.User{
		{Id: 1, Username: "sales", Password: "hash", AffCode: "s001", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 10, Username: "customer-a", Password: "hash", AffCode: "c010", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
		{Id: 11, Username: "customer-b", Password: "hash", AffCode: "c011", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
		{Id: 12, Username: "customer-c", Password: "hash", AffCode: "c012", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
	}
	require.NoError(t, model.DB.Create(&users).Error)
	require.NoError(t, model.AssignManagedCustomers(1, []int{10, 11, 12}, 99))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/sales/1/customers?p=2&page_size=2", nil)
	context.Params = gin.Params{{Key: "id", Value: "1"}}
	GetSalesCustomers(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"total":3`)
	require.Contains(t, recorder.Body.String(), `"username":"customer-a"`)
	require.NotContains(t, recorder.Body.String(), "customer-b")
	require.NotContains(t, recorder.Body.String(), "customer-c")
}

func TestGetAvailableSalesCustomersOnlyExcludesCurrentSalesAssignments(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withManagedLogControllerDatabase(t)

	users := []model.User{
		{Id: 1, Username: "sales-a", Password: "hash", AffCode: "s001", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 2, Username: "sales-b", Password: "hash", AffCode: "s002", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 10, Username: "customer-a", Password: "hash", AffCode: "c010", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
		{Id: 11, Username: "customer-b", Password: "hash", AffCode: "c011", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
	}
	require.NoError(t, model.DB.Create(&users).Error)
	require.NoError(t, model.AssignManagedCustomers(1, []int{10}, 99))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/sales/available-customers?manager_user_id=2&p=1&page_size=20", nil)
	GetAvailableSalesCustomers(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"total":2`)
	require.Contains(t, recorder.Body.String(), `"username":"customer-a"`)
	require.Contains(t, recorder.Body.String(), `"username":"customer-b"`)

	recorder = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/sales/available-customers?manager_user_id=1&p=1&page_size=20", nil)
	GetAvailableSalesCustomers(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"total":1`)
	require.NotContains(t, recorder.Body.String(), `"username":"customer-a"`)
	require.Contains(t, recorder.Body.String(), `"username":"customer-b"`)
}

func TestRemoveSalesCustomersRemovesBatchAtomically(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withManagedLogControllerDatabase(t)

	users := []model.User{
		{Id: 1, Username: "sales", Password: "hash", AffCode: "s001", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 10, Username: "customer-a", Password: "hash", AffCode: "c010", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
		{Id: 11, Username: "customer-b", Password: "hash", AffCode: "c011", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
	}
	require.NoError(t, model.DB.Create(&users).Error)
	require.NoError(t, model.AssignManagedCustomers(1, []int{10, 11}, 99))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/sales/1/customers/remove",
		strings.NewReader(`{"customer_user_ids":[10,11]}`),
	)
	context.Params = gin.Params{{Key: "id", Value: "1"}}
	RemoveSalesCustomers(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)
	var relationCount int64
	require.NoError(t, model.DB.Model(&model.ManagedUserRelation{}).Count(&relationCount).Error)
	require.Zero(t, relationCount)

	require.NoError(t, model.AssignManagedCustomers(1, []int{10, 11}, 99))
	recorder = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/sales/1/customers/remove",
		strings.NewReader(`{"customer_user_ids":[10,999]}`),
	)
	context.Params = gin.Params{{Key: "id", Value: "1"}}
	RemoveSalesCustomers(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":false`)
	require.NoError(t, model.DB.Model(&model.ManagedUserRelation{}).Count(&relationCount).Error)
	require.Equal(t, int64(2), relationCount)
}

func performUpdateUserControllerRequest(t *testing.T, operatorRole int, body string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/user/", strings.NewReader(body))
	context.Set("id", 99)
	context.Set("username", "operator")
	context.Set("role", operatorRole)
	UpdateUser(context)
	return recorder
}

func TestAdminCanSwitchCommonUserToSalesButCannotPromoteToAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withManagedLogControllerDatabase(t)

	users := []model.User{
		{Id: 1, Username: "sales", Password: "hash", AffCode: "s001", Role: common.RoleSalesUser, Status: common.UserStatusEnabled},
		{Id: 10, Username: "customer", DisplayName: "Customer", Password: "hash", AffCode: "c010", Group: "default", Role: common.RoleCommonUser, Status: common.UserStatusEnabled},
	}
	require.NoError(t, model.DB.Create(&users).Error)
	require.NoError(t, model.AssignManagedCustomers(1, []int{10}, 99))

	promoteToSales := performUpdateUserControllerRequest(
		t,
		common.RoleAdminUser,
		`{"id":10,"username":"customer","display_name":"Customer","group":"default","role":5}`,
	)
	require.Equal(t, http.StatusOK, promoteToSales.Code)
	require.Contains(t, promoteToSales.Body.String(), `"success":true`)

	var savedCustomer model.User
	require.NoError(t, model.DB.First(&savedCustomer, 10).Error)
	require.Equal(t, common.RoleSalesUser, savedCustomer.Role)
	var relationCount int64
	require.NoError(t, model.DB.Model(&model.ManagedUserRelation{}).Count(&relationCount).Error)
	require.Zero(t, relationCount)

	promoteToAdmin := performUpdateUserControllerRequest(
		t,
		common.RoleAdminUser,
		`{"id":10,"username":"customer","display_name":"Customer","group":"default","role":10}`,
	)
	require.Contains(t, promoteToAdmin.Body.String(), `"success":false`)
	require.NoError(t, model.DB.First(&savedCustomer, 10).Error)
	require.Equal(t, common.RoleSalesUser, savedCustomer.Role)
}
