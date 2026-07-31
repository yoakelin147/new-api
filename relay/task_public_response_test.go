package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestTaskModel2PublicDtoPreservesResultsAndRemovesInternalData(t *testing.T) {
	task := &model.Task{
		ID:         91,
		CreatedAt:  100,
		UpdatedAt:  200,
		TaskID:     "task_public",
		Platform:   constant.TaskPlatform("sora"),
		UserId:     7,
		Group:      "premium",
		ChannelId:  12,
		Quota:      345,
		Action:     constant.TaskActionImageGenerate,
		Status:     model.TaskStatusSuccess,
		FailReason: "",
		SubmitTime: 110,
		StartTime:  120,
		FinishTime: 190,
		Progress:   "100%",
		Properties: model.Properties{
			Input:             "draw a lighthouse",
			UpstreamModelName: "provider-private-model",
			OriginModelName:   "customer-model",
		},
		PrivateData: model.TaskPrivateData{
			Key:            "private-key",
			UpstreamTaskID: "upstream-task-123",
			ResultURL:      "https://example.com/result.png",
		},
		Data: []byte(`{
			"id":"upstream-task-123",
			"task_id":"upstream-task-123",
			"model":"provider-private-model",
			"status":"completed",
			"channel_id":12,
			"userId":7,
			"group":"premium",
			"quota":345,
			"platform":"sora",
			"api_key":"private-key",
			"private_data":{"upstream_task_id":"upstream-task-123"},
			"data":{
				"id":"upstream-task-123",
				"model_name":"provider-private-model",
				"images":[{"id":"media-1","url":"https://example.com/result.png"}],
				"operation":{"name":"projects/internal-project/locations/us-central1/models/provider-private-model/operations/op-1"}
			}
		}`),
	}
	originalData := append([]byte(nil), task.Data...)

	got := TaskModel2PublicDto(task)

	require.Equal(t, originalData, []byte(task.Data))

	var data map[string]any
	require.NoError(t, common.Unmarshal(got.Data, &data))
	wantData := map[string]any{
		"id":      "task_public",
		"model":   "customer-model",
		"status":  "completed",
		"task_id": "task_public",
		"data": map[string]any{
			"id":         "task_public",
			"model_name": "customer-model",
			"images": []any{
				map[string]any{
					"id":  "media-1",
					"url": "https://example.com/result.png",
				},
			},
			"operation": map[string]any{
				"name": "task_public",
			},
		},
	}
	require.Equal(t, wantData, data)
	wantDataJSON, err := common.Marshal(wantData)
	require.NoError(t, err)
	require.Equal(t, &dto.PublicTaskDto{
		CreatedAt:  100,
		UpdatedAt:  200,
		TaskID:     "task_public",
		Action:     constant.TaskActionImageGenerate,
		Status:     string(model.TaskStatusSuccess),
		FailReason: "",
		ResultURL:  "https://example.com/result.png",
		SubmitTime: 110,
		StartTime:  120,
		FinishTime: 190,
		Progress:   "100%",
		Properties: dto.PublicTaskProperties{
			Input:           "draw a lighthouse",
			OriginModelName: "customer-model",
		},
		Data: wantDataJSON,
	}, got)

	responseJSON, err := common.Marshal(got)
	require.NoError(t, err)
	for _, privateValue := range []string{
		`"id":91`,
		`"user_id"`,
		`"channel_id"`,
		`"group"`,
		`"quota"`,
		`"platform"`,
		`"upstream_model_name"`,
		"provider-private-model",
		"upstream-task-123",
		"private-key",
		"internal-project",
	} {
		require.NotContains(t, string(responseJSON), privateValue)
	}
}

func TestTaskModel2PublicImageDtoIsCompact(t *testing.T) {
	task := &model.Task{
		ID:         91,
		CreatedAt:  100,
		UpdatedAt:  200,
		TaskID:     "task_public",
		Platform:   constant.TaskPlatform("sora"),
		UserId:     7,
		Group:      "premium",
		ChannelId:  12,
		Quota:      345,
		Action:     constant.TaskActionImageGenerate,
		Status:     model.TaskStatusSuccess,
		FailReason: "https://example.com/legacy-result.png",
		SubmitTime: 110,
		StartTime:  120,
		FinishTime: 190,
		Progress:   "100%",
		Properties: model.Properties{
			Input:             "draw a lighthouse",
			UpstreamModelName: "provider-private-model",
			OriginModelName:   "customer-model",
		},
		PrivateData: model.TaskPrivateData{
			Key:            "private-key",
			UpstreamTaskID: "upstream-task-123",
			ResultURL:      "https://example.com/result.png",
		},
		Data: []byte(`{"id":"upstream-task-123","model":"provider-private-model"}`),
	}

	got := TaskModel2PublicImageDto(task)

	require.Equal(t, &dto.PublicImageTaskDto{
		TaskID:      "task_public",
		Status:      string(model.TaskStatusSuccess),
		Progress:    "100%",
		Model:       "customer-model",
		ResultURL:   "https://example.com/result.png",
		CreatedAt:   100,
		CompletedAt: 190,
	}, got)
	responseJSON, err := common.Marshal(dto.TaskResponse[any]{
		Code: "success",
		Data: got,
	})
	require.NoError(t, err)
	require.JSONEq(t, `{
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
	}`, string(responseJSON))
	for _, removedField := range []string{
		`"id"`,
		`"updated_at"`,
		`"action"`,
		`"submit_time"`,
		`"start_time"`,
		`"finish_time"`,
		`"properties"`,
		`"data":{"id"`,
		"provider-private-model",
		"upstream-task-123",
		"private-key",
	} {
		require.NotContains(t, string(responseJSON), removedField)
	}
}

func TestTaskModel2PublicImageDtoKeepsFailureSeparateFromResultURL(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_public",
		Status:     model.TaskStatusFailure,
		Progress:   "100%",
		FailReason: "generation failed",
	}

	got := TaskModel2PublicImageDto(task)

	require.Equal(t, &dto.PublicImageTaskDto{
		TaskID:     "task_public",
		Status:     string(model.TaskStatusFailure),
		Progress:   "100%",
		FailReason: "generation failed",
	}, got)
}

func TestTaskModel2PublicDtoDoesNotChangeInternalTaskConversion(t *testing.T) {
	task := &model.Task{
		ID:        91,
		TaskID:    "task_public",
		Platform:  constant.TaskPlatform("sora"),
		UserId:    7,
		Group:     "premium",
		ChannelId: 12,
		Quota:     345,
		Properties: model.Properties{
			UpstreamModelName: "provider-private-model",
			OriginModelName:   "customer-model",
		},
	}

	got := TaskModel2Dto(task)

	require.Equal(t, &dto.TaskDto{
		ID:         91,
		TaskID:     "task_public",
		Platform:   "sora",
		UserId:     7,
		Group:      "premium",
		ChannelId:  12,
		Quota:      345,
		Properties: task.Properties,
	}, got)
}

func TestTaskModel2PublicDtoDropsMalformedProviderData(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public",
		Data:   []byte(`{"api_key":"private-key"`),
	}

	got := TaskModel2PublicDto(task)
	responseJSON, err := common.Marshal(got)

	require.NoError(t, err)
	require.JSONEq(t, `{
		"created_at":0,
		"updated_at":0,
		"task_id":"task_public",
		"action":"",
		"status":"",
		"fail_reason":"",
		"submit_time":0,
		"start_time":0,
		"finish_time":0,
		"progress":"",
		"properties":{"input":""},
		"data":null
	}`, string(responseJSON))
	require.NotContains(t, string(responseJSON), "private-key")
}
