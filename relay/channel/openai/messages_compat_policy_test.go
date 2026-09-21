package openai

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestOpenAIChannelStillConvertsMessagesWhenMoonshotPolicyIsDisabled(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	original := settings.MessagesToChatCompletionsPolicy
	settings.MessagesToChatCompletionsPolicy = model_setting.ChatCompletionsToResponsesPolicy{}
	t.Cleanup(func() { settings.MessagesToChatCompletionsPolicy = original })

	info := &relaycommon.RelayInfo{
		RelayFormat:     types.RelayFormatClaude,
		OriginModelName: "gpt-4o",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenAI,
			UpstreamModelName: "gpt-4o",
		},
	}
	request := &dto.ClaudeRequest{
		Model:    "gpt-4o",
		Messages: []dto.ClaudeMessage{{Role: "user", Content: "hello"}},
	}

	converted, err := (&Adaptor{}).ConvertClaudeRequest(nil, info, request)

	require.NoError(t, err)
	chatRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "hello", chatRequest.Messages[0].StringContent())
}
