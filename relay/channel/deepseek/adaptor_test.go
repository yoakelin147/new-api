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

func TestConvertOpenAIResponsesRequestDefaultDisabledUsesNativeResponses(t *testing.T) {
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

	request := dto.OpenAIResponsesRequest{
		Model: "deepseek-chat",
		Input: inputJSON,
	}
	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, deepSeekResponsesRelayInfo(), request)

	require.NoError(t, err)
	nativeRequest, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	require.Equal(t, request, nativeRequest)
}

func TestGetRequestURLForNativeResponses(t *testing.T) {
	info := deepSeekResponsesRelayInfo()
	info.ChannelBaseUrl = "https://api.deepseek.com"

	got, err := (&Adaptor{}).GetRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://api.deepseek.com/responses", got)
}

func TestConvertOpenAIResponsesRequestAppliesV4ThinkingSuffix(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	oldPolicy := settings.ResponsesToChatCompletionsPolicy
	settings.ResponsesToChatCompletionsPolicy = model_setting.ChatCompletionsToResponsesPolicy{}
	t.Cleanup(func() {
		settings.ResponsesToChatCompletionsPolicy = oldPolicy
	})

	info := deepSeekResponsesRelayInfo()
	info.UpstreamModelName = "deepseek-v4-flash-max"
	request := dto.OpenAIResponsesRequest{Model: "deepseek-v4-flash-max"}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.NoError(t, err)
	nativeRequest, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	require.Equal(t, "deepseek-v4-flash", nativeRequest.Model)
	require.NotNil(t, nativeRequest.Reasoning)
	require.Equal(t, "max", nativeRequest.Reasoning.Effort)
	require.Equal(t, "deepseek-v4-flash", info.UpstreamModelName)
	require.Equal(t, "max", info.ReasoningEffort)
}

func TestConvertOpenAIResponsesRequestMapsDisabledV4Thinking(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	oldPolicy := settings.ResponsesToChatCompletionsPolicy
	settings.ResponsesToChatCompletionsPolicy = model_setting.ChatCompletionsToResponsesPolicy{}
	t.Cleanup(func() {
		settings.ResponsesToChatCompletionsPolicy = oldPolicy
	})

	info := deepSeekResponsesRelayInfo()
	info.UpstreamModelName = "deepseek-v4-flash-none"
	request := dto.OpenAIResponsesRequest{Model: "deepseek-v4-flash-none"}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.NoError(t, err)
	nativeRequest, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	require.Equal(t, "deepseek-v4-flash", nativeRequest.Model)
	require.NotNil(t, nativeRequest.Reasoning)
	require.Equal(t, "none", nativeRequest.Reasoning.Effort)
	require.Equal(t, "none", info.ReasoningEffort)
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
			ChannelBaseUrl:    "https://api.deepseek.com",
			UpstreamModelName: "deepseek-chat",
		},
	}
}
