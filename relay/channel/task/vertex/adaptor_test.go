package vertex

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/stretchr/testify/require"
)

func TestConvertToOpenAIVideoUsesCustomerFacingModel(t *testing.T) {
	upstreamOperation := "projects/internal-project/locations/us-central1/models/provider-private-model/operations/op-1"
	task := &model.Task{
		TaskID:    "task_public",
		Status:    model.TaskStatusSuccess,
		Progress:  "100%",
		CreatedAt: 10,
		UpdatedAt: 20,
		Properties: model.Properties{
			UpstreamModelName: "provider-private-model",
			OriginModelName:   "customer-model",
		},
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: taskcommon.EncodeLocalTaskID(upstreamOperation),
		},
	}

	body, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)

	require.NoError(t, err)
	var got dto.OpenAIVideo
	require.NoError(t, common.Unmarshal(body, &got))
	require.Equal(t, dto.OpenAIVideo{
		ID:          "task_public",
		Object:      "video",
		Model:       "customer-model",
		Status:      dto.VideoStatusCompleted,
		Progress:    100,
		CreatedAt:   10,
		CompletedAt: 20,
	}, got)
	require.NotContains(t, string(body), "provider-private-model")
	require.NotContains(t, string(body), "internal-project")
}
