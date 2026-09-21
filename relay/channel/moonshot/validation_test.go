package moonshot

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestValidateKimiK3MessagesRequest(t *testing.T) {
	info := moonshotRelayInfo(types.RelayFormatClaude)
	info.ApiType = constant.APITypeMoonshot
	info.UpstreamModelName = "kimi-k3"

	tests := []struct {
		name    string
		request *dto.ClaudeRequest
		wantErr string
	}{
		{name: "missing max tokens", request: &dto.ClaudeRequest{}, wantErr: "max_tokens is required"},
		{name: "output limit", request: &dto.ClaudeRequest{MaxTokens: common.GetPointer[uint](1048577)}, wantErr: "1048576"},
		{name: "too many stop sequences", request: &dto.ClaudeRequest{MaxTokens: common.GetPointer[uint](1), StopSequences: []string{"1", "2", "3", "4", "5", "6"}}, wantErr: "at most 5"},
		{name: "long stop sequence", request: &dto.ClaudeRequest{MaxTokens: common.GetPointer[uint](1), StopSequences: []string{strings.Repeat("a", 33)}}, wantErr: "32 bytes"},
		{name: "invalid effort", request: &dto.ClaudeRequest{MaxTokens: common.GetPointer[uint](1), OutputConfig: []byte(`{"effort":"medium"}`)}, wantErr: "low, high, max"},
		{name: "invalid tool choice", request: &dto.ClaudeRequest{MaxTokens: common.GetPointer[uint](1), ToolChoice: map[string]any{"type": "required"}}, wantErr: "auto, any, none"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateRequest(info, test.request)
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestValidateKimiK3ResponsesRequest(t *testing.T) {
	info := moonshotRelayInfo(types.RelayFormatOpenAIResponses)
	info.ApiType = constant.APITypeMoonshot
	info.UpstreamModelName = "kimi-k3"

	tests := []struct {
		name    string
		request *dto.OpenAIResponsesRequest
		wantErr string
	}{
		{name: "output limit", request: &dto.OpenAIResponsesRequest{MaxOutputTokens: common.GetPointer[uint](1048577)}, wantErr: "1048576"},
		{name: "invalid effort", request: &dto.OpenAIResponsesRequest{Reasoning: &dto.Reasoning{Effort: "medium"}}, wantErr: "low, high, max"},
		{name: "invalid tool choice", request: &dto.OpenAIResponsesRequest{ToolChoice: []byte(`"required"`)}, wantErr: "must be auto"},
		{name: "invalid cache ttl", request: &dto.OpenAIResponsesRequest{PromptCacheOptions: []byte(`{"mode":"implicit","ttl":"2h"}`)}, wantErr: "5m, 1h"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateRequest(info, test.request)
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestValidateRequestDoesNotApplyToOtherChannels(t *testing.T) {
	info := moonshotRelayInfo(types.RelayFormatOpenAIResponses)
	info.ApiType = constant.APITypeOpenAI
	info.UpstreamModelName = "kimi-k3"

	err := ValidateRequest(info, &dto.OpenAIResponsesRequest{
		Reasoning: &dto.Reasoning{Effort: "medium"},
	})

	require.NoError(t, err)
}
