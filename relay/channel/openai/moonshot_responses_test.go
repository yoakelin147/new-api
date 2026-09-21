package openai

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestMoonshotResponsesCompatibilityMetadata(t *testing.T) {
	info := &relaycommon.RelayInfo{
		Request:     &dto.OpenAIResponsesRequest{PromptCacheOptions: []byte(`{"mode":"implicit","ttl":"1h"}`)},
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeMoonshot},
	}
	response := &dto.OpenAIResponsesResponse{Usage: &dto.Usage{
		PromptTokens: 2, CompletionTokens: 3, TotalTokens: 5, InputTokens: 2, OutputTokens: 3,
		InputTokensDetails: &dto.InputTokenDetails{},
	}}
	applyMoonshotResponsesMetadata(info, response)

	data, err := marshalResponsesCompatibilityPayload(info, response)
	require.NoError(t, err)
	require.Contains(t, string(data), `"prompt_cache_options":{"mode":"implicit","ttl":"1h"}`)
	require.Contains(t, string(data), `"cache_write_tokens":0`)
	require.NotContains(t, string(data), `"prompt_tokens"`)
	require.NotContains(t, string(data), `"completion_tokens"`)
	require.Contains(t, string(data), `"output_tokens_details"`)
	require.NotContains(t, string(data), `"completion_tokens_details"`)
	require.NotContains(t, string(data), `"claude_cache_creation_5_m_tokens"`)
}

func TestMoonshotResponsesStreamSequenceStartsAtZero(t *testing.T) {
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeMoonshot}}
	next := 0
	first := &dto.ResponsesStreamResponse{Type: "response.created"}
	second := &dto.ResponsesStreamResponse{Type: "response.completed"}

	applyMoonshotStreamMetadata(info, first, &next)
	applyMoonshotStreamMetadata(info, second, &next)

	require.Equal(t, 0, *first.SequenceNumber)
	require.Equal(t, 1, *second.SequenceNumber)
	require.Equal(t, 2, next)
}
