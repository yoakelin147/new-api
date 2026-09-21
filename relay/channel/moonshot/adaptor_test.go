package moonshot

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
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenAIResponsesRequestUsesChatCompatibilityPolicy(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	oldPolicy := settings.ResponsesToChatCompletionsPolicy
	settings.ResponsesToChatCompletionsPolicy = model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       true,
		ChannelIDs:    []int{25},
		ModelPatterns: []string{"^kimi-k2.5$"},
	}
	t.Cleanup(func() { settings.ResponsesToChatCompletionsPolicy = oldPolicy })

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	input, err := common.Marshal("hello")
	require.NoError(t, err)
	info := moonshotRelayInfo(types.RelayFormatOpenAIResponses)
	info.RelayMode = relayconstant.RelayModeResponses
	info.RequestURLPath = "/v1/responses"

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(c, info, dto.OpenAIResponsesRequest{
		Model: "kimi-k2.5",
		Input: input,
	})

	require.NoError(t, err)
	chatRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "hello", chatRequest.Messages[0].StringContent())
	require.Equal(t, "/v1/chat/completions", info.RequestURLPath)
	require.True(t, c.GetBool(ginKeyResponsesViaChatCompletions))
	require.Equal(t, types.RelayFormatOpenAI, info.GetFinalRequestRelayFormat())
}

func TestConvertOpenAIResponsesRequestKeepsNativeProtocolByDefault(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	oldPolicy := settings.ResponsesToChatCompletionsPolicy
	settings.ResponsesToChatCompletionsPolicy = model_setting.ChatCompletionsToResponsesPolicy{}
	t.Cleanup(func() { settings.ResponsesToChatCompletionsPolicy = oldPolicy })

	info := moonshotRelayInfo(types.RelayFormatOpenAIResponses)
	info.RelayMode = relayconstant.RelayModeResponses
	info.RequestURLPath = "/v1/responses"
	input, err := common.Marshal("hello")
	require.NoError(t, err)

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{Model: "kimi-k2.5", Input: input})
	require.NoError(t, err)
	relaycommon.AppendRequestConversionFromRequest(info, converted)

	_, ok := converted.(*dto.OpenAIResponsesRequest)
	require.True(t, ok)
	requestURL, err := (&Adaptor{}).GetRequestURL(info)
	require.NoError(t, err)
	require.Equal(t, "https://api.moonshot.cn/v1/responses", requestURL)
}

func TestConvertOpenAIResponsesRequestForwardsKimiPromptCacheOptions(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	oldPolicy := settings.ResponsesToChatCompletionsPolicy
	settings.ResponsesToChatCompletionsPolicy = model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       true,
		ChannelIDs:    []int{25},
		ModelPatterns: []string{"^kimi-k3$"},
	}
	t.Cleanup(func() { settings.ResponsesToChatCompletionsPolicy = oldPolicy })

	info := moonshotRelayInfo(types.RelayFormatOpenAIResponses)
	info.OriginModelName = "kimi-k3"
	info.UpstreamModelName = "kimi-k3"
	info.RelayMode = relayconstant.RelayModeResponses
	input, err := common.Marshal("hello")
	require.NoError(t, err)

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model:              "kimi-k3",
		Input:              input,
		PromptCacheOptions: []byte(`{"mode":"implicit","ttl":"1h"}`),
	})
	require.NoError(t, err)
	data, err := common.Marshal(converted)
	require.NoError(t, err)
	require.Contains(t, string(data), `"prompt_cache_options":{"mode":"implicit","ttl":"1h"}`)
}

func TestConvertClaudeRequestUsesOpenAICompatibilityForRegularMoonshot(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	oldPolicy := settings.MessagesToChatCompletionsPolicy
	settings.MessagesToChatCompletionsPolicy = model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       true,
		ChannelIDs:    []int{25},
		ModelPatterns: []string{"^kimi-k2.5$"},
	}
	t.Cleanup(func() { settings.MessagesToChatCompletionsPolicy = oldPolicy })

	info := moonshotRelayInfo(types.RelayFormatClaude)
	request := &dto.ClaudeRequest{Model: "kimi-k2.5"}
	request.Messages = []dto.ClaudeMessage{{Role: "user", Content: "hello"}}

	converted, err := (&Adaptor{}).ConvertClaudeRequest(nil, info, request)
	require.NoError(t, err)
	relaycommon.AppendRequestConversionFromRequest(info, converted)

	chatRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "hello", chatRequest.Messages[0].StringContent())
	requestURL, err := (&Adaptor{}).GetRequestURL(info)
	require.NoError(t, err)
	require.Equal(t, "https://api.moonshot.cn/v1/chat/completions", requestURL)
}

func TestConvertClaudeRequestMapsKimiK3MessagesOptions(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	oldPolicy := settings.MessagesToChatCompletionsPolicy
	settings.MessagesToChatCompletionsPolicy = model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       true,
		ChannelIDs:    []int{25},
		ModelPatterns: []string{"^kimi-k3$"},
	}
	t.Cleanup(func() { settings.MessagesToChatCompletionsPolicy = oldPolicy })

	info := moonshotRelayInfo(types.RelayFormatClaude)
	info.OriginModelName = "kimi-k3"
	info.UpstreamModelName = "kimi-k3"
	request := &dto.ClaudeRequest{
		Model:        "kimi-k3",
		MaxTokens:    common.GetPointer[uint](1024),
		OutputConfig: []byte(`{"effort":"high","format":{"type":"json_schema","schema":{"type":"object"}}}`),
		ToolChoice:   map[string]any{"type": "none"},
		Messages:     []dto.ClaudeMessage{{Role: "user", Content: "hello"}},
	}

	converted, err := (&Adaptor{}).ConvertClaudeRequest(nil, info, request)
	require.NoError(t, err)
	chatRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "high", chatRequest.ReasoningEffort)
	require.Equal(t, "none", chatRequest.ToolChoice)
	require.NotNil(t, chatRequest.ResponseFormat)
	require.Equal(t, "json_schema", chatRequest.ResponseFormat.Type)
	require.JSONEq(t, `{"name":"response","schema":{"type":"object"},"strict":true}`, string(chatRequest.ResponseFormat.JsonSchema))
}

func TestConvertClaudeRequestMapsAnyToolChoiceToRequired(t *testing.T) {
	target := &dto.GeneralOpenAIRequest{}
	err := ApplyKimiMessagesCompatibility(&dto.ClaudeRequest{
		ToolChoice: map[string]any{"type": "any"},
	}, target)

	require.NoError(t, err)
	require.Equal(t, "required", target.ToolChoice)
}

func TestConvertClaudeRequestKeepsNativeProtocolByDefault(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	oldPolicy := settings.MessagesToChatCompletionsPolicy
	settings.MessagesToChatCompletionsPolicy = model_setting.ChatCompletionsToResponsesPolicy{}
	t.Cleanup(func() { settings.MessagesToChatCompletionsPolicy = oldPolicy })

	info := moonshotRelayInfo(types.RelayFormatClaude)
	request := &dto.ClaudeRequest{Model: "kimi-k2.5"}
	request.Messages = []dto.ClaudeMessage{{Role: "user", Content: "hello"}}

	converted, err := (&Adaptor{}).ConvertClaudeRequest(nil, info, request)
	require.NoError(t, err)
	relaycommon.AppendRequestConversionFromRequest(info, converted)

	_, ok := converted.(*dto.ClaudeRequest)
	require.True(t, ok)
	requestURL, err := (&Adaptor{}).GetRequestURL(info)
	require.NoError(t, err)
	require.Equal(t, "https://api.moonshot.cn/anthropic/v1/messages", requestURL)
}

func TestNativeMessagesDoesNotHardcodeAliyunEndpoint(t *testing.T) {
	info := moonshotRelayInfo(types.RelayFormatClaude)
	info.ChannelBaseUrl = "https://workspace.ap-southeast-1.maas.aliyuncs.com/compatible-mode"
	info.RequestURLPath = "/v1/messages"

	requestURL, err := (&Adaptor{}).GetRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://workspace.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1/messages", requestURL)
}

func TestNativeMessagesUsesClientPathForOtherCustomBase(t *testing.T) {
	info := moonshotRelayInfo(types.RelayFormatClaude)
	info.ChannelBaseUrl = "https://provider.example"
	info.RequestURLPath = "/v1/messages"

	requestURL, err := (&Adaptor{}).GetRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://provider.example/v1/messages", requestURL)
}

func TestConvertClaudeRequestKeepsNativeKimiCodingPlan(t *testing.T) {
	info := moonshotRelayInfo(types.RelayFormatClaude)
	info.ChannelBaseUrl = "kimi-coding-plan"
	request := &dto.ClaudeRequest{Model: "kimi-for-coding"}

	converted, err := (&Adaptor{}).ConvertClaudeRequest(nil, info, request)
	require.NoError(t, err)
	relaycommon.AppendRequestConversionFromRequest(info, converted)

	_, ok := converted.(*dto.ClaudeRequest)
	require.True(t, ok)
	requestURL, err := (&Adaptor{}).GetRequestURL(info)
	require.NoError(t, err)
	require.Equal(t, "https://api.kimi.com/coding/v1/messages", requestURL)
}

func moonshotRelayInfo(format types.RelayFormat) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RelayFormat:     format,
		OriginModelName: "kimi-k2.5",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         25,
			ChannelType:       constant.ChannelTypeMoonshot,
			ChannelBaseUrl:    "https://api.moonshot.cn",
			UpstreamModelName: "kimi-k2.5",
		},
	}
}

func TestConvertOpenAIRequestKimiK26UsesOnlyAllowedTemperature(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:       "kimi-k2.6",
		Temperature: common.GetPointer[float64](0.7),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.6",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)

	require.NoError(t, err)
	convertedRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.NotNil(t, convertedRequest.Temperature)
	require.Equal(t, 1.0, *convertedRequest.Temperature)
}

func TestConvertOpenAIRequestKimiK26KeepsOmittedTemperatureOmitted(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model: "kimi-k2.6",
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.6",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)

	require.NoError(t, err)
	convertedRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Nil(t, convertedRequest.Temperature)
}

func TestConvertOpenAIRequestOtherMoonshotModelKeepsTemperature(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:       "kimi-k2.5",
		Temperature: common.GetPointer[float64](0.7),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.5",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)

	require.NoError(t, err)
	convertedRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.NotNil(t, convertedRequest.Temperature)
	require.Equal(t, 0.7, *convertedRequest.Temperature)
}
