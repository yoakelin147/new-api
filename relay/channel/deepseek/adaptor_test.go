package deepseek

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenAIResponsesRequestDefaultDisabledKeepsNotImplemented(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	oldPolicy := settings.ResponsesToChatCompletionsPolicy
	settings.ResponsesToChatCompletionsPolicy = model_setting.ChatCompletionsToResponsesPolicy{
		Enabled: false,
	}
	t.Cleanup(func() {
		settings.ResponsesToChatCompletionsPolicy = oldPolicy
	})

	inputJSON, err := common.Marshal("hello")
	require.NoError(t, err)

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, deepSeekResponsesRelayInfo(), dto.OpenAIResponsesRequest{
		Model: "deepseek-chat",
		Input: inputJSON,
	})

	require.Nil(t, converted)
	require.EqualError(t, err, "not implemented")
}

func TestConvertOpenAIResponsesRequestUsesChatPolicy(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	oldPolicy := settings.ResponsesToChatCompletionsPolicy
	settings.ResponsesToChatCompletionsPolicy = model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       true,
		AllChannels:   false,
		ChannelIDs:    []int{5},
		ModelPatterns: []string{"^deepseek-chat$"},
	}
	t.Cleanup(func() {
		settings.ResponsesToChatCompletionsPolicy = oldPolicy
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	inputJSON, err := common.Marshal("hello")
	require.NoError(t, err)

	info := deepSeekResponsesRelayInfo()
	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(c, info, dto.OpenAIResponsesRequest{
		Model: "deepseek-chat",
		Input: inputJSON,
	})
	require.NoError(t, err)

	chatReq, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "/v1/chat/completions", info.RequestURLPath)
	require.True(t, c.GetBool(ginKeyResponsesViaChatCompletions))
	require.Equal(t, "deepseek-chat", chatReq.Model)
	require.Len(t, chatReq.Messages, 1)
	require.Equal(t, "user", chatReq.Messages[0].Role)
	require.Equal(t, "hello", chatReq.Messages[0].StringContent())
}

func deepSeekResponsesRelayInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName: "deepseek-chat",
		RequestURLPath:  "/v1/responses",
		RelayMode:       relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         5,
			ChannelType:       constant.ChannelTypeDeepSeek,
			UpstreamModelName: "deepseek-chat",
		},
	}
}
