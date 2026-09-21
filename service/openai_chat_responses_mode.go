package service

import (
	"github.com/QuantumNous/new-api/service/relayconvert"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func ShouldChatCompletionsUseResponsesPolicy(policy model_setting.ChatCompletionsToResponsesPolicy, channelID int, channelType int, model string) bool {
	return relayconvert.ShouldChatCompletionsUseResponsesPolicy(policy, channelID, channelType, model)
}

func ShouldChatCompletionsUseResponsesGlobal(channelID int, channelType int, model string) bool {
	return relayconvert.ShouldChatCompletionsUseResponsesGlobal(channelID, channelType, model)
}

func ShouldResponsesUseChatCompletionsGlobal(channelID int, channelType int, model string) bool {
	return relayconvert.ShouldResponsesUseChatCompletionsGlobal(channelID, channelType, model)
}

func ShouldMessagesUseChatCompletionsGlobal(channelID int, channelType int, model string) bool {
	return relayconvert.ShouldMessagesUseChatCompletionsGlobal(channelID, channelType, model)
}
