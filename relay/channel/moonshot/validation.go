package moonshot

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

const kimiK3MaxOutputTokens = uint(1048576)

func ValidateRequest(info *relaycommon.RelayInfo, request dto.Request) error {
	if info == nil || info.ChannelMeta == nil || info.ApiType != constant.APITypeMoonshot {
		return nil
	}

	if !isKimiK3OrLater(info.UpstreamModelName) && !isKimiK3OrLater(info.OriginModelName) {
		return nil
	}

	switch req := request.(type) {
	case *dto.ClaudeRequest:
		return validateKimiMessagesRequest(req)
	case *dto.OpenAIResponsesRequest:
		return validateKimiResponsesRequest(req)
	default:
		return nil
	}
}

func validateKimiMessagesRequest(request *dto.ClaudeRequest) error {
	if request.MaxTokens == nil || *request.MaxTokens == 0 {
		return errors.New("max_tokens is required")
	}
	if *request.MaxTokens > kimiK3MaxOutputTokens {
		return fmt.Errorf("max_tokens must not exceed %d", kimiK3MaxOutputTokens)
	}
	if len(request.StopSequences) > 5 {
		return errors.New("stop_sequences must contain at most 5 items")
	}
	for _, stop := range request.StopSequences {
		if len([]byte(stop)) > 32 {
			return errors.New("each stop_sequences item must be at most 32 bytes")
		}
	}

	if len(request.OutputConfig) > 0 {
		var config kimiMessagesOutputConfig
		if err := common.Unmarshal(request.OutputConfig, &config); err != nil {
			return fmt.Errorf("invalid output_config: %w", err)
		}
		if config.Effort != "" && !isKimiK3ReasoningEffort(config.Effort) {
			return errors.New("output_config.effort must be one of low, high, max")
		}
		if config.Format != nil && (config.Format.Type != "json_schema" || config.Format.Schema == nil) {
			return errors.New("output_config.format requires type json_schema and schema")
		}
	}

	if request.ToolChoice != nil {
		raw, err := common.Marshal(request.ToolChoice)
		if err != nil {
			return fmt.Errorf("invalid tool_choice: %w", err)
		}
		var choice dto.ClaudeToolChoice
		if err := common.Unmarshal(raw, &choice); err != nil {
			return fmt.Errorf("invalid tool_choice: %w", err)
		}
		switch strings.TrimSpace(choice.Type) {
		case "auto", "any", "none":
		default:
			return errors.New("tool_choice.type must be one of auto, any, none")
		}
	}
	return nil
}

func validateKimiResponsesRequest(request *dto.OpenAIResponsesRequest) error {
	if request.MaxOutputTokens != nil && *request.MaxOutputTokens > kimiK3MaxOutputTokens {
		return fmt.Errorf("max_output_tokens must not exceed %d", kimiK3MaxOutputTokens)
	}
	if request.Reasoning != nil && request.Reasoning.Effort != "" && !isKimiK3ReasoningEffort(request.Reasoning.Effort) {
		return errors.New("reasoning.effort must be one of low, high, max")
	}
	if len(request.ToolChoice) > 0 && common.GetJsonType(request.ToolChoice) != "null" {
		var choice string
		if common.GetJsonType(request.ToolChoice) != "string" || common.Unmarshal(request.ToolChoice, &choice) != nil || choice != "auto" {
			return errors.New("tool_choice must be auto for Kimi Responses API")
		}
	}
	if len(request.PromptCacheOptions) > 0 && common.GetJsonType(request.PromptCacheOptions) != "null" {
		var options struct {
			Mode string `json:"mode"`
			TTL  string `json:"ttl"`
		}
		if err := common.Unmarshal(request.PromptCacheOptions, &options); err != nil {
			return fmt.Errorf("invalid prompt_cache_options: %w", err)
		}
		if options.Mode != "" && options.Mode != "implicit" {
			return errors.New("prompt_cache_options.mode must be implicit")
		}
		if options.TTL != "" && options.TTL != "5m" && options.TTL != "1h" {
			return errors.New("prompt_cache_options.ttl must be one of 5m, 1h")
		}
	}
	return nil
}

func isKimiK3ReasoningEffort(effort string) bool {
	switch effort {
	case "low", "high", "max":
		return true
	default:
		return false
	}
}
