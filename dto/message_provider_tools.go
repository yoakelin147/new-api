package dto

import (
	"encoding/json"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/tidwall/sjson"
)

func (r GeneralOpenAIRequest) MarshalJSON() ([]byte, error) {
	type alias GeneralOpenAIRequest
	data, err := common.Marshal(alias(r))
	if err != nil || !r.KimiDynamicToolsEnabled {
		return data, err
	}
	for i, message := range r.Messages {
		if len(message.ProviderTools) == 0 {
			continue
		}
		data, err = sjson.SetRawBytes(data, "messages."+strconv.Itoa(i)+".tools", message.ProviderTools)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (r *GeneralOpenAIRequest) UnmarshalJSON(data []byte) error {
	type alias GeneralOpenAIRequest
	var decoded alias
	if err := common.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = GeneralOpenAIRequest(decoded)
	var extension struct {
		Messages []struct {
			Tools json.RawMessage `json:"tools"`
		} `json:"messages"`
	}
	if err := common.Unmarshal(data, &extension); err != nil {
		return err
	}
	for i := range r.Messages {
		if i < len(extension.Messages) {
			r.Messages[i].ProviderTools = extension.Messages[i].Tools
		}
	}
	return nil
}
