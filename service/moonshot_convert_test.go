package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestResponseOpenAI2ClaudePreservesMoonshotReasoning(t *testing.T) {
	response := &dto.OpenAITextResponse{
		Choices: []dto.OpenAITextResponseChoice{{
			Message:      dto.Message{Role: "assistant", Content: "answer", ReasoningContent: common.GetPointer("reasoning")},
			FinishReason: "stop",
		}},
		Usage: dto.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			PromptTokensDetails: dto.InputTokenDetails{
				CachedTokens:     4,
				CacheWriteTokens: 2,
			},
			CompletionTokenDetails: dto.OutputTokenDetails{ReasoningTokens: 3},
		},
	}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeMoonshot}}

	converted := ResponseOpenAI2Claude(response, info)

	require.Len(t, converted.Content, 2)
	require.Equal(t, "thinking", converted.Content[0].Type)
	require.Equal(t, "reasoning", *converted.Content[0].Thinking)
	require.NotNil(t, converted.Content[0].Signature)
	require.Empty(t, *converted.Content[0].Signature)
	require.Equal(t, "text", converted.Content[1].Type)
	require.Equal(t, "answer", *converted.Content[1].Text)
	require.Equal(t, 3, converted.Usage.OutputTokensDetails.ThinkingTokens)
	require.Equal(t, 4, converted.Usage.InputTokens)
	require.Equal(t, 4, converted.Usage.CacheReadInputTokens)
	require.Equal(t, 2, converted.Usage.CacheCreationInputTokens)
	data, err := common.Marshal(converted)
	require.NoError(t, err)
	require.NotContains(t, string(data), "claude_cache_creation_5_m_tokens")
	require.NotContains(t, string(data), "claude_cache_creation_1_h_tokens")
}

func TestResponseOpenAI2ClaudeKeepsLegacyUsageFieldsForOtherChannels(t *testing.T) {
	converted := ResponseOpenAI2Claude(&dto.OpenAITextResponse{
		Choices: []dto.OpenAITextResponseChoice{{
			Message: dto.Message{Content: "answer", ReasoningContent: common.GetPointer("reasoning")},
		}},
	}, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI},
	})

	require.Len(t, converted.Content, 1)
	require.Equal(t, "text", converted.Content[0].Type)
	data, err := common.Marshal(converted)
	require.NoError(t, err)
	require.Contains(t, string(data), "claude_cache_creation_5_m_tokens")
	require.Contains(t, string(data), "claude_cache_creation_1_h_tokens")
}
