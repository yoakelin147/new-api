package moonshot

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

var kimiModelPattern = regexp.MustCompile(`(?i)^kimi-k([0-9]+)(?:[.-][a-z0-9][a-z0-9.-]*)?$`)
var kimiToolNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]{0,255}$`)

// 后续 K 系列复用消息级工具扩展；未来新字段仍需按上游协议单独评估。
func isKimiK3OrLater(model string) bool {
	model = strings.TrimSpace(model)
	if slash := strings.LastIndexByte(model, '/'); slash >= 0 {
		model = model[slash+1:]
	}
	match := kimiModelPattern.FindStringSubmatch(model)
	if match == nil {
		return false
	}
	major, err := strconv.ParseUint(match[1], 10, 32)
	return err == nil && major >= 3
}

func convertKimiRequest(_ *gin.Context, request *dto.GeneralOpenAIRequest) (any, error) {
	if err := validateKimiDynamicTools(request); err != nil {
		return nil, types.WithOpenAIError(types.OpenAIError{
			Message: err.Error(), Type: "invalid_request_error", Code: string(types.ErrorCodeInvalidRequest),
		}, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	if err := normalizeKimiThinking(request); err != nil {
		return nil, types.WithOpenAIError(types.OpenAIError{
			Message: err.Error(), Type: "invalid_request_error", Code: string(types.ErrorCodeInvalidRequest),
		}, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	request.KimiDynamicToolsEnabled = true
	return request, nil
}

func normalizeKimiThinking(request *dto.GeneralOpenAIRequest) error {
	if len(request.THINKING) == 0 || request.ReasoningEffort != "" || common.GetJsonType(request.THINKING) != "object" {
		return nil
	}
	var thinking map[string]json.RawMessage
	if err := common.Unmarshal(request.THINKING, &thinking); err != nil {
		return fmt.Errorf("invalid thinking parameter: %w", err)
	}
	if _, exists := thinking["effort"]; exists {
		return nil
	}
	if rawType, exists := thinking["type"]; exists {
		var thinkingType string
		if err := common.Unmarshal(rawType, &thinkingType); err != nil || thinkingType != "enabled" {
			return nil
		}
	}
	effort, err := common.Marshal("max")
	if err != nil {
		return err
	}
	thinking["effort"] = effort
	request.THINKING, err = common.Marshal(thinking)
	return err
}

func validateKimiDynamicTools(request *dto.GeneralOpenAIRequest) error {
	names := make(map[string]struct{}, len(request.Tools))
	for _, tool := range request.Tools {
		if tool.Function.Name != "" {
			names[tool.Function.Name] = struct{}{}
		}
	}
	for index, message := range request.Messages {
		if len(message.ProviderTools) == 0 {
			continue
		}
		if message.Role != "system" {
			return fmt.Errorf("messages[%d].tools is only allowed in system messages", index)
		}
		if message.Content != nil {
			text, isText := message.Content.(string)
			items, isArray := message.Content.([]any)
			if !(isText && text == "") && !(isArray && len(items) == 0) {
				return fmt.Errorf("messages[%d].content must be empty with dynamic tools", index)
			}
		}
		if common.GetJsonType(message.ProviderTools) != "array" {
			return fmt.Errorf("messages[%d].tools must be an array", index)
		}
		var tools []map[string]any
		if err := common.Unmarshal(message.ProviderTools, &tools); err != nil {
			return fmt.Errorf("messages[%d].tools must contain objects", index)
		}
		for _, tool := range tools {
			if tool == nil || tool["type"] != "function" {
				return errors.New("dynamic tool must be an object with type function")
			}
			function, ok := tool["function"].(map[string]any)
			if !ok {
				return errors.New("dynamic tool function must be an object")
			}
			name, ok := function["name"].(string)
			if !ok || !kimiToolNamePattern.MatchString(name) {
				return errors.New("dynamic tool name must start with a letter or underscore and contain at most 256 letters, digits, underscores or hyphens")
			}
			if _, exists := names[name]; exists {
				return errors.New("duplicate tool name in global or dynamic tools")
			}
			names[name] = struct{}{}
		}
	}
	return nil
}
