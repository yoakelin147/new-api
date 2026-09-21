package openai

import (
	"encoding/json"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func applyMoonshotResponsesMetadata(info *relaycommon.RelayInfo, response *dto.OpenAIResponsesResponse) {
	if info == nil || info.ChannelType != constant.ChannelTypeMoonshot || response == nil {
		return
	}
	response.PromptCacheOptions = json.RawMessage(`{"mode":"implicit","ttl":"5m"}`)
	if request, ok := info.Request.(*dto.OpenAIResponsesRequest); ok && len(request.PromptCacheOptions) > 0 {
		response.PromptCacheOptions = append(json.RawMessage(nil), request.PromptCacheOptions...)
	}
}

func applyMoonshotStreamMetadata(info *relaycommon.RelayInfo, payload *dto.ResponsesStreamResponse, next *int) {
	if info == nil || info.ChannelType != constant.ChannelTypeMoonshot || payload == nil || next == nil {
		return
	}
	payload.SequenceNumber = common.GetPointer(*next)
	*next = *next + 1
	applyMoonshotResponsesMetadata(info, payload.Response)
}

func marshalResponsesCompatibilityPayload(info *relaycommon.RelayInfo, value any) ([]byte, error) {
	data, err := common.Marshal(value)
	if err != nil || info == nil || info.ChannelType != constant.ChannelTypeMoonshot {
		return data, err
	}

	var root map[string]any
	if err := common.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	normalizeMoonshotResponsesObject(root)
	if response, ok := root["response"].(map[string]any); ok {
		normalizeMoonshotResponsesObject(response)
	}
	return common.Marshal(root)
}

func normalizeMoonshotResponsesObject(response map[string]any) {
	usage, ok := response["usage"].(map[string]any)
	if !ok {
		return
	}
	delete(usage, "prompt_tokens")
	delete(usage, "completion_tokens")
	delete(usage, "prompt_tokens_details")
	delete(usage, "claude_cache_creation_5_m_tokens")
	delete(usage, "claude_cache_creation_1_h_tokens")
	if details, ok := usage["input_tokens_details"].(map[string]any); ok {
		if _, exists := details["cache_write_tokens"]; !exists {
			details["cache_write_tokens"] = 0
		}
		delete(details, "text_tokens")
		delete(details, "audio_tokens")
		delete(details, "image_tokens")
	}
	if details, ok := usage["completion_tokens_details"].(map[string]any); ok {
		delete(details, "text_tokens")
		delete(details, "audio_tokens")
		delete(details, "image_tokens")
		usage["output_tokens_details"] = details
		delete(usage, "completion_tokens_details")
	}
}
