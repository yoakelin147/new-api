package relayconvert

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/stretchr/testify/require"
)

func TestShouldMessagesUseChatCompletionsGlobalDefaultsDisabled(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	original := settings.MessagesToChatCompletionsPolicy
	settings.MessagesToChatCompletionsPolicy = model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:     false,
		AllChannels: true,
	}
	t.Cleanup(func() { settings.MessagesToChatCompletionsPolicy = original })

	require.False(t, ShouldMessagesUseChatCompletionsGlobal(25, 25, "kimi-k3"))
}

func TestShouldMessagesUseChatCompletionsGlobalUsesConfiguredScope(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	original := settings.MessagesToChatCompletionsPolicy
	settings.MessagesToChatCompletionsPolicy = model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       true,
		ChannelIDs:    []int{25},
		ModelPatterns: []string{"^kimi-k3$"},
	}
	t.Cleanup(func() { settings.MessagesToChatCompletionsPolicy = original })

	require.True(t, ShouldMessagesUseChatCompletionsGlobal(25, 25, "kimi-k3"))
	require.False(t, ShouldMessagesUseChatCompletionsGlobal(26, 25, "kimi-k3"))
	require.False(t, ShouldMessagesUseChatCompletionsGlobal(25, 25, "kimi-k2.5"))
}
