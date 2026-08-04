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
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.ManagedUserRelation{}, &model.Log{}))
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
	require.NoError(t, model.AssignManagedCustomers(2, []int{11}, 99))
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
