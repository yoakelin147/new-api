package controller

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type systemAccessKeyResponse struct {
	Success bool                    `json:"success"`
	Message string                  `json:"message"`
	Data    []model.SystemAccessKey `json:"data"`
}

type createdSystemAccessKeyResponse struct {
	Success bool                       `json:"success"`
	Message string                     `json:"message"`
	Data    createdSystemAccessKeyData `json:"data"`
}

type createdSystemAccessKeyData struct {
	AccessKey model.SystemAccessKey `json:"access_key"`
	Key       string                `json:"key"`
}

func resetControllerSystemAccessKeys(t *testing.T) {
	t.Helper()
	previousDB := model.DB
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = testDB
	require.NoError(t, model.DB.AutoMigrate(&model.SystemAccessKey{}))
	t.Cleanup(func() {
		model.DB = previousDB
	})
}

func performSystemAccessKeyControllerRequest(
	t *testing.T,
	method string,
	path string,
	body string,
	handler gin.HandlerFunc,
) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("id", 99)
	if id := context.Param("id"); id == "" {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) > 0 {
			context.Params = append(context.Params, gin.Param{Key: "id", Value: parts[len(parts)-1]})
		}
	}
	handler(context)
	return recorder
}

func TestMonitoringAccessKeyControllerLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetControllerSystemAccessKeys(t)

	createRecorder := performSystemAccessKeyControllerRequest(
		t,
		http.MethodPost,
		"/api/system-access-key/monitoring",
		`{"name":" Zabbix production ","expires_at":null}`,
		CreateMonitoringAccessKey,
	)
	require.Equal(t, http.StatusOK, createRecorder.Code)
	var created createdSystemAccessKeyResponse
	require.NoError(t, common.Unmarshal(createRecorder.Body.Bytes(), &created))
	require.True(t, created.Success)
	require.Equal(t, "Zabbix production", created.Data.AccessKey.Name)
	require.Nil(t, created.Data.AccessKey.ExpiresAt)
	require.Len(t, created.Data.Key, 52)
	require.True(t, strings.HasPrefix(created.Data.Key, "sak_"))
	require.Equal(t, created.Data.Key[:systemAccessKeyPrefixLength], created.Data.AccessKey.KeyPrefix)
	require.Empty(t, created.Data.AccessKey.KeyHash)

	stored, err := model.GetSystemAccessKeyByHash(
		constant.SystemAccessKeyTypeMonitoring,
		model.HashSystemAccessKey(created.Data.Key),
	)
	require.NoError(t, err)
	require.NotEmpty(t, stored.KeyHash)
	require.Equal(t, 99, stored.CreatedBy)

	listRecorder := performSystemAccessKeyControllerRequest(
		t,
		http.MethodGet,
		"/api/system-access-key/monitoring",
		"",
		ListMonitoringAccessKeys,
	)
	require.Equal(t, http.StatusOK, listRecorder.Code)
	require.NotContains(t, listRecorder.Body.String(), created.Data.Key)
	require.NotContains(t, listRecorder.Body.String(), stored.KeyHash)
	var listed systemAccessKeyResponse
	require.NoError(t, common.Unmarshal(listRecorder.Body.Bytes(), &listed))
	require.True(t, listed.Success)
	require.Len(t, listed.Data, 1)

	disableRecorder := performSystemAccessKeyControllerRequest(
		t,
		http.MethodPatch,
		"/api/system-access-key/monitoring/"+strconv.Itoa(stored.Id),
		`{"enabled":false}`,
		UpdateMonitoringAccessKeyStatus,
	)
	require.Equal(t, http.StatusOK, disableRecorder.Code)
	stored, err = model.GetSystemAccessKeyByHash(
		constant.SystemAccessKeyTypeMonitoring,
		model.HashSystemAccessKey(created.Data.Key),
	)
	require.NoError(t, err)
	require.Equal(t, constant.SystemAccessKeyStatusDisabled, stored.Status)

	deleteRecorder := performSystemAccessKeyControllerRequest(
		t,
		http.MethodDelete,
		"/api/system-access-key/monitoring/"+strconv.Itoa(stored.Id),
		"",
		DeleteMonitoringAccessKey,
	)
	require.Equal(t, http.StatusOK, deleteRecorder.Code)
	_, err = model.GetSystemAccessKeyByHash(
		constant.SystemAccessKeyTypeMonitoring,
		model.HashSystemAccessKey(created.Data.Key),
	)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestCreateMonitoringAccessKeyValidatesExpiration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetControllerSystemAccessKeys(t)

	past := common.GetTimestamp() - 1
	recorder := performSystemAccessKeyControllerRequest(
		t,
		http.MethodPost,
		"/api/system-access-key/monitoring",
		`{"name":"expired","expires_at":`+strconv.FormatInt(past, 10)+`}`,
		CreateMonitoringAccessKey,
	)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestUpdateMonitoringAccessKeyStatusRejectsExpiredKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetControllerSystemAccessKeys(t)

	expiresAt := common.GetTimestamp() - 1
	key := &model.SystemAccessKey{
		Name:        "expired",
		KeyType:     constant.SystemAccessKeyTypeMonitoring,
		KeyHash:     model.HashSystemAccessKey("sak_expired"),
		KeyPrefix:   "sak_expired",
		Status:      constant.SystemAccessKeyStatusDisabled,
		ExpiresAt:   &expiresAt,
		CreatedTime: expiresAt - 1,
		UpdatedTime: expiresAt - 1,
	}
	require.NoError(t, model.CreateSystemAccessKey(key))

	recorder := performSystemAccessKeyControllerRequest(
		t,
		http.MethodPatch,
		"/api/system-access-key/monitoring/"+strconv.Itoa(key.Id),
		`{"enabled":true}`,
		UpdateMonitoringAccessKeyStatus,
	)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
