package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func performRoleAuthRequest(t *testing.T, sessionRole int, currentRole int, authMiddleware func() func(c *gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	previousDB := model.DB
	testDB, err := gorm.Open(
		sqlite.Open("file:sales-auth-"+strconv.FormatInt(time.Now().UnixNano(), 10)+"?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	require.NoError(t, err)
	model.DB = testDB
	require.NoError(t, testDB.AutoMigrate(&model.User{}))
	require.NoError(t, testDB.Create(&model.User{
		Id: 42, Username: "role-user", Password: "hash", AffCode: "r042",
		Role: currentRole, Status: common.UserStatusEnabled,
	}).Error)
	t.Cleanup(func() { model.DB = previousDB })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("sales-auth-test"))))
	router.GET("/login", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("username", "role-user")
		session.Set("role", sessionRole)
		session.Set("id", 42)
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "default")
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	router.GET("/protected", authMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	loginRecorder := httptest.NewRecorder()
	router.ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodGet, "/login", nil))
	require.Equal(t, http.StatusNoContent, loginRecorder.Code)

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("New-Api-User", strconv.Itoa(42))
	for _, sessionCookie := range loginRecorder.Result().Cookies() {
		request.AddCookie(sessionCookie)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestSalesAuthAllowsOnlySalesRole(t *testing.T) {
	salesRecorder := performRoleAuthRequest(t, common.RoleSalesUser, common.RoleSalesUser, SalesAuth)
	require.Contains(t, salesRecorder.Body.String(), `"success":true`)

	commonUserRecorder := performRoleAuthRequest(t, common.RoleCommonUser, common.RoleCommonUser, SalesAuth)
	require.Contains(t, commonUserRecorder.Body.String(), `"success":false`)

	adminRecorder := performRoleAuthRequest(t, common.RoleAdminUser, common.RoleAdminUser, SalesAuth)
	require.Contains(t, adminRecorder.Body.String(), `"success":false`)

	demotedRecorder := performRoleAuthRequest(t, common.RoleSalesUser, common.RoleCommonUser, SalesAuth)
	require.Contains(t, demotedRecorder.Body.String(), `"success":false`)
}

func TestAdministrativeAuthRejectsStaleElevatedSession(t *testing.T) {
	adminRecorder := performRoleAuthRequest(t, common.RoleAdminUser, common.RoleAdminUser, AdminAuth)
	require.Contains(t, adminRecorder.Body.String(), `"success":true`)

	rootRecorder := performRoleAuthRequest(t, common.RoleRootUser, common.RoleRootUser, AdminAuth)
	require.Contains(t, rootRecorder.Body.String(), `"success":true`)

	demotedAdminRecorder := performRoleAuthRequest(t, common.RoleAdminUser, common.RoleSalesUser, AdminAuth)
	require.Contains(t, demotedAdminRecorder.Body.String(), `"success":false`)

	demotedRootRecorder := performRoleAuthRequest(t, common.RoleRootUser, common.RoleAdminUser, RootAuth)
	require.Contains(t, demotedRootRecorder.Body.String(), `"success":false`)
}
