package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

type sunoFailureAdaptor struct{}

func (*sunoFailureAdaptor) Init(*relaycommon.RelayInfo)                                    {}
func (*sunoFailureAdaptor) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error)          { return nil, nil }
func (*sunoFailureAdaptor) AdjustBillingOnComplete(*model.Task, *relaycommon.TaskInfo) int { return 0 }
func (*sunoFailureAdaptor) FetchTask(_ string, _ string, body map[string]any, _ string) (*http.Response, error) {
	ids, _ := body["ids"].([]string)
	items := make([]dto.SunoDataResponse, 0, len(ids))
	for _, id := range ids {
		items = append(items, dto.SunoDataResponse{
			TaskID:     id,
			Status:     string(model.TaskStatusFailure),
			FailReason: "upstream failed",
			FinishTime: time.Now().Unix(),
		})
	}
	bodyBytes, err := common.Marshal(dto.TaskResponse[[]dto.SunoDataResponse]{Code: dto.TaskSuccessCode, Data: items})
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(bodyBytes))}, nil
}

func TestUpdateSunoTasksRefundsOnlyCASWinner(t *testing.T) {
	truncate(t)
	const userID, tokenID, channelID = 901, 901, 901
	seedUser(t, userID, 10_000)
	seedToken(t, tokenID, userID, "suno-cas-token", 6_000)
	baseURL := "https://suno.test"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id: channelID, Type: constant.ChannelTypeSunoAPI, Name: "suno-cas", Key: "channel-key",
		Status: common.ChannelStatusEnabled, BaseURL: &baseURL,
	}).Error)
	model.InitChannelCache()

	task := makeTask(userID, channelID, 2_500, tokenID, BillingSourceWallet, 0)
	task.TaskID = "suno-public-cas"
	task.Platform = constant.TaskPlatformSuno
	task.Status = model.TaskStatusInProgress
	task.PrivateData.UpstreamTaskID = "suno-upstream-cas"
	require.NoError(t, model.DB.Create(task).Error)

	var first, stale model.Task
	require.NoError(t, model.DB.First(&first, task.ID).Error)
	require.NoError(t, model.DB.First(&stale, task.ID).Error)
	previousAdaptor := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return &sunoFailureAdaptor{} }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousAdaptor })

	require.NoError(t, UpdateSunoTasks(context.Background(), map[int][]string{channelID: {"suno-upstream-cas"}}, map[string]*model.Task{"suno-upstream-cas": &first}))
	require.NoError(t, UpdateSunoTasks(context.Background(), map[int][]string{channelID: {"suno-upstream-cas"}}, map[string]*model.Task{"suno-upstream-cas": &stale}))

	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	require.Equal(t, string(model.TaskStatusFailure), string(reloaded.Status))
	require.Equal(t, 10_000+2_500, getUserQuota(t, userID))
	require.Equal(t, 6_000+2_500, getTokenRemainQuota(t, tokenID))
	require.Equal(t, int64(1), countLogs(t))
}
