package moonshot

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

type kimiMessagesOutputConfig struct {
	Effort string                    `json:"effort"`
	Format *kimiMessagesOutputFormat `json:"format"`
}

type kimiMessagesOutputFormat struct {
	Type   string `json:"type"`
	Schema any    `json:"schema"`
}

func ApplyKimiMessagesCompatibility(source *dto.ClaudeRequest, target *dto.GeneralOpenAIRequest) error {
	if source == nil || target == nil {
		return errors.New("request is nil")
	}

	if len(source.OutputConfig) > 0 {
		var outputConfig kimiMessagesOutputConfig
		if err := common.Unmarshal(source.OutputConfig, &outputConfig); err != nil {
			return fmt.Errorf("invalid output_config: %w", err)
		}
		target.ReasoningEffort = outputConfig.Effort
		if outputConfig.Format != nil {
			if outputConfig.Format.Type != "json_schema" || outputConfig.Format.Schema == nil {
				return errors.New("output_config.format requires type json_schema and schema")
			}
			jsonSchema, err := common.Marshal(dto.FormatJsonSchema{
				Name:   "response",
				Schema: outputConfig.Format.Schema,
				Strict: []byte("true"),
			})
			if err != nil {
				return err
			}
			target.ResponseFormat = &dto.ResponseFormat{Type: "json_schema", JsonSchema: jsonSchema}
		}
	}

	if source.ToolChoice != nil {
		raw, err := common.Marshal(source.ToolChoice)
		if err != nil {
			return fmt.Errorf("invalid tool_choice: %w", err)
		}
		var toolChoice dto.ClaudeToolChoice
		if err := common.Unmarshal(raw, &toolChoice); err != nil {
			return fmt.Errorf("invalid tool_choice: %w", err)
		}
		switch strings.TrimSpace(toolChoice.Type) {
		case "auto", "none":
			target.ToolChoice = toolChoice.Type
		case "any":
			target.ToolChoice = "required"
		default:
			return fmt.Errorf("unsupported tool_choice.type %q", toolChoice.Type)
		}
	}

	return nil
}
