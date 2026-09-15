package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestExportAllLogsReturnsEveryMatchingLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB, previousLogDB := model.DB, model.LOG_DB
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:log-export-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&model.Log{}))
	t.Cleanup(func() { model.DB, model.LOG_DB = previousDB, previousLogDB })

	logs := make([]model.Log, 0, 10_002)
	for i := range 10_001 {
		logs = append(logs, model.Log{
			UserId:       i + 1,
			CreatedAt:    1_700_000_000 + int64(i),
			Type:         model.LogTypeConsume,
			Username:     "customer",
			ModelName:    "gpt-export",
			RequestId:    fmt.Sprintf("request-%03d", i),
			TokenName:    "customer-token",
			PromptTokens: i,
		})
	}
	logs = append(logs, model.Log{
		UserId: 999, CreatedAt: 1_700_010_000, Type: model.LogTypeConsume,
		Username: "other", ModelName: "gpt-export", RequestId: "not-exported",
	})
	require.NoError(t, db.CreateInBatches(&logs, 500).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/log/export?username=customer&model_name=gpt-export&start_timestamp=1700000000&end_timestamp=1700020000",
		nil,
	)

	ExportAllLogs(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.Contains(t, recorder.Body.String(), `"request_id":"request-000"`)
	require.Contains(t, recorder.Body.String(), `"request_id":"request-10000"`)
	require.NotContains(t, recorder.Body.String(), "not-exported")
	require.Equal(t, 10_001, strings.Count(recorder.Body.String(), `"request_id":"request-`))
}
