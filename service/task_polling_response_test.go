package service

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestParseNewAPIImageTaskResponse(t *testing.T) {
	responseBody := []byte(`{
		"code":"success",
		"message":"",
		"data":{
			"task_id":"task_public",
			"status":"SUCCESS",
			"progress":"100%",
			"model":"customer-model",
			"result_url":"https://example.com/result.png",
			"created_at":100,
			"completed_at":190
		}
	}`)

	got, ok := parseNewAPIImageTaskResponse(responseBody)

	require.True(t, ok)
	require.Equal(t, &relaycommon.TaskInfo{
		TaskID:   "task_public",
		Status:   "SUCCESS",
		Url:      "https://example.com/result.png",
		Progress: "100%",
	}, got)
}

func TestParseNewAPIImageTaskResponseRejectsNonSuccessEnvelope(t *testing.T) {
	got, ok := parseNewAPIImageTaskResponse([]byte(`{"code":"error","data":{}}`))

	require.False(t, ok)
	require.Nil(t, got)
}
