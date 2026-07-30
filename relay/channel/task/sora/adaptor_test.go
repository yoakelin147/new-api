package sora

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParseTaskResultExtractsTuziStyleVideoURL(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"id":"upstream_task_1",
		"task_id":"upstream_task_1",
		"object":"video",
		"model":"gpt-image-2",
		"status":"completed",
		"progress":100,
		"video_url":"https://example.com/result.png"
	}`)

	taskInfo, err := adaptor.ParseTaskResult(body)

	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusSuccess), taskInfo.Status)
	require.Equal(t, "100%", taskInfo.Progress)
	require.Equal(t, "https://example.com/result.png", taskInfo.Url)
}

func TestParseTaskResultMapsFailureReason(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"id":"upstream_task_1",
		"status":"failed",
		"error":{"message":"upstream failed","code":"task_failed"}
	}`)

	taskInfo, err := adaptor.ParseTaskResult(body)

	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusFailure), taskInfo.Status)
	require.Equal(t, "upstream failed", taskInfo.Reason)
}

func TestConvertToOpenAIVideoReturnsPublicTaskID(t *testing.T) {
	adaptor := &TaskAdaptor{}
	task := &model.Task{
		TaskID:     "task_local",
		Status:     model.TaskStatusSuccess,
		Progress:   "100%",
		CreatedAt:  10,
		UpdatedAt:  20,
		SubmitTime: 10,
		FinishTime: 20,
		Properties: model.Properties{
			OriginModelName: "gpt-image-2",
		},
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "task_upstream",
			ResultURL:      "https://example.com/result.png",
		},
		Data: []byte(`{
			"id":"task_upstream",
			"task_id":"task_upstream",
			"status":"completed",
			"video_url":"https://example.com/result.png"
		}`),
	}

	body, err := adaptor.ConvertToOpenAIVideo(task)

	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, common.Unmarshal(body, &got))
	require.Equal(t, "task_local", got["id"])
	require.Equal(t, "task_local", got["task_id"])
	require.Equal(t, "completed", got["status"])
	require.Equal(t, "https://example.com/result.png", got["video_url"])
	require.NotContains(t, string(body), "task_upstream")
}

func TestExtractTaskResultURLSupportsNestedImageURL(t *testing.T) {
	body := []byte(`{
		"data": {
			"result": {
				"images": [
					{"url": ["https://example.com/image.png"]}
				]
			}
		}
	}`)

	require.Equal(t, "https://example.com/image.png", extractTaskResultURL(body))
}

func TestBuildRequestBodyAppliesConfiguredParameterMappingAndPreservesType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	templatesJSON, err := common.Marshal([]dto.AsyncTaskUpstreamTemplate{
		{
			Enabled:     true,
			ChannelID:   19,
			RequestPath: "/v1/images/generations/async",
			Model:       "gpt-image-2",
			ParameterMappings: []dto.AsyncTaskParameterMapping{
				{
					When: map[string]any{
						"size":       "1:1",
						"resolution": "2k",
					},
					StripFields: []string{"resolution"},
					SetFields: map[string]any{
						"size": "2048x2048",
						"type": "image",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	previousTemplates := common.OptionMap[setting.AsyncTaskUpstreamTemplatesOptionKey]
	common.OptionMap[setting.AsyncTaskUpstreamTemplatesOptionKey] = string(templatesJSON)
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap[setting.AsyncTaskUpstreamTemplatesOptionKey] = previousTemplates
		common.OptionMapRWMutex.Unlock()
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	requestBody := `{"model":"gpt-image-2","prompt":"draw a fox","size":"1:1","resolution":"2k"}`
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations/async", bytes.NewBufferString(requestBody))
	context.Request.Header.Set("Content-Type", "application/json")
	t.Cleanup(func() {
		common.CleanupBodyStorage(context)
	})

	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		RequestURLPath:  "/v1/images/generations/async",
		OriginModelName: "gpt-image-2",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         19,
			ChannelType:       1,
			UpstreamModelName: "gpt-image-2-upstream",
		},
	}

	reader, err := adaptor.BuildRequestBody(context, info)
	require.NoError(t, err)
	actualBody, err := io.ReadAll(reader)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, common.Unmarshal(actualBody, &got))
	require.Equal(t, map[string]any{
		"model":  "gpt-image-2",
		"prompt": "draw a fox",
		"size":   "2048x2048",
		"type":   "image",
	}, got)
}

func TestDoResponseUsesConfiguredSubmitTaskIDMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	templatesJSON, err := common.Marshal([]dto.AsyncTaskUpstreamTemplate{
		{
			Enabled:     true,
			ChannelID:   19,
			RequestPath: "/v1/images/generations/async",
			Model:       "gpt-image-2",
			ResponseMapping: &dto.AsyncTaskResponseMapping{
				SubmitTaskIDPaths: []string{"data.task_id"},
				SubmitStatusPaths: []string{"data.status"},
			},
		},
	})
	require.NoError(t, err)
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	previousTemplates := common.OptionMap[setting.AsyncTaskUpstreamTemplatesOptionKey]
	common.OptionMap[setting.AsyncTaskUpstreamTemplatesOptionKey] = string(templatesJSON)
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap[setting.AsyncTaskUpstreamTemplatesOptionKey] = previousTemplates
		common.OptionMapRWMutex.Unlock()
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	adaptor := &TaskAdaptor{}
	resp := &http.Response{
		Body: io.NopCloser(bytes.NewReader([]byte(`{
			"id": "wrong_id",
			"data": {"task_id": "mapped_upstream_task", "status": "submitted"}
		}`))),
	}
	info := &relaycommon.RelayInfo{
		RequestURLPath:  "/v1/images/generations/async",
		OriginModelName: "gpt-image-2",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action:       constant.TaskActionImageGenerate,
			PublicTaskID: "task_public",
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:   19,
			ChannelType: 1,
		},
	}

	taskID, _, taskErr := adaptor.DoResponse(context, resp, info)

	require.Nil(t, taskErr)
	require.Equal(t, "mapped_upstream_task", taskID)
	require.Contains(t, recorder.Body.String(), `"id":"task_public"`)
	require.Contains(t, recorder.Body.String(), `"task_id":"task_public"`)
	require.Contains(t, recorder.Body.String(), `"object":"image"`)
	require.Contains(t, recorder.Body.String(), `"model":"gpt-image-2"`)
	require.Contains(t, recorder.Body.String(), `"status":"queued"`)
	require.NotContains(t, recorder.Body.String(), `"created_at":0`)
}

func TestParseTaskResultUsesConfiguredResponseMapping(t *testing.T) {
	adaptor := &TaskAdaptor{
		responseMapping: &dto.AsyncTaskResponseMapping{
			FetchStatusPaths:    []string{"data.state"},
			FetchResultURLPaths: []string{"data.assets.0.src"},
			FetchProgressPaths:  []string{"data.percent"},
			FetchStatusMap: map[string]string{
				"done": "SUCCESS",
			},
		},
	}
	body := []byte(`{
		"data": {
			"state": "done",
			"percent": 98.5,
			"assets": [{"src": "https://example.com/result.png"}]
		}
	}`)

	taskInfo, err := adaptor.ParseTaskResult(body)

	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusSuccess), taskInfo.Status)
	require.Equal(t, "98.50%", taskInfo.Progress)
	require.Equal(t, "https://example.com/result.png", taskInfo.Url)
}

func TestEstimateBillingSkipsImageGenerateRatios(t *testing.T) {
	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: constant.TaskActionImageGenerate,
		},
	}
	info.Action = constant.TaskActionImageGenerate

	require.Nil(t, adaptor.EstimateBilling(nil, info))
}
