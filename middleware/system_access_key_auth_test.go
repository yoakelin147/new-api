package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const testSystemAccessKey = "sak_0123456789abcdef0123456789abcdef"

func performSystemAccessKeyRequest(t *testing.T, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/metrics", SystemAccessKeyAuth(constant.SystemAccessKeyTypeMonitoring), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func resetSystemAccessKeys(t *testing.T) {
	t.Helper()
	previousDB := model.DB
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = testDB
	t.Cleanup(func() {
		model.DB = previousDB
	})
	require.NoError(t, model.DB.AutoMigrate(&model.SystemAccessKey{}))
	require.NoError(t, model.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&model.SystemAccessKey{}).Error)
	t.Cleanup(func() {
		require.NoError(t, model.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&model.SystemAccessKey{}).Error)
	})
}

func TestSystemAccessKeyAuthRejectsMissingInvalidDisabledAndExpiredKeys(t *testing.T) {
	resetSystemAccessKeys(t)

	recorder := performSystemAccessKeyRequest(t, "")
	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	recorder = performSystemAccessKeyRequest(t, "Basic "+testSystemAccessKey)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	recorder = performSystemAccessKeyRequest(t, "Bearer wrong")
	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	now := int64(1)
	disabled := &model.SystemAccessKey{
		Name:        "disabled",
		KeyType:     constant.SystemAccessKeyTypeMonitoring,
		KeyHash:     model.HashSystemAccessKey(testSystemAccessKey),
		KeyPrefix:   "sak_01234567",
		Status:      constant.SystemAccessKeyStatusDisabled,
		CreatedTime: now,
		UpdatedTime: now,
	}
	require.NoError(t, model.CreateSystemAccessKey(disabled))
	recorder = performSystemAccessKeyRequest(t, "Bearer "+testSystemAccessKey)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	expiredRawKey := "sak_expired-system-access-key"
	expiresAt := int64(1)
	expired := &model.SystemAccessKey{
		Name:        "expired",
		KeyType:     constant.SystemAccessKeyTypeMonitoring,
		KeyHash:     model.HashSystemAccessKey(expiredRawKey),
		KeyPrefix:   "sak_expired",
		Status:      constant.SystemAccessKeyStatusEnabled,
		ExpiresAt:   &expiresAt,
		CreatedTime: 1,
		UpdatedTime: 1,
	}
	require.NoError(t, model.CreateSystemAccessKey(expired))
	recorder = performSystemAccessKeyRequest(t, "Bearer "+expiredRawKey)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestSystemAccessKeyAuthAllowsEnabledKey(t *testing.T) {
	resetSystemAccessKeys(t)
	now := int64(4102444800)
	key := &model.SystemAccessKey{
		Name:        "zabbix",
		KeyType:     constant.SystemAccessKeyTypeMonitoring,
		KeyHash:     model.HashSystemAccessKey(testSystemAccessKey),
		KeyPrefix:   "sak_01234567",
		Status:      constant.SystemAccessKeyStatusEnabled,
		ExpiresAt:   &now,
		CreatedTime: 1,
		UpdatedTime: 1,
	}
	require.NoError(t, model.CreateSystemAccessKey(key))

	recorder := performSystemAccessKeyRequest(t, "Bearer "+testSystemAccessKey)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
}
