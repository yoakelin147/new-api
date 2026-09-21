package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestChannelValidateRequestPathRewriteSettings(t *testing.T) {
	validSetting := `{"request_path_rewrite":[{"conditions":{"request_path":"/v1/messages"},"target_path":"/apps/anthropic/v1/messages"}]}`
	channel := &Channel{Setting: common.GetPointer(validSetting)}

	require.NoError(t, channel.ValidateSettings())

	invalidCondition := `{"request_path_rewrite":[{"conditions":{"request_path":"/v1/messages","client_type":"sdk"},"target_path":"/messages"}]}`
	channel.Setting = common.GetPointer(invalidCondition)
	require.ErrorContains(t, channel.ValidateSettings(), "unsupported condition")

	invalidTarget := `{"request_path_rewrite":[{"conditions":{"request_path":"/v1/messages"},"target_path":"https://attacker.example/messages"}]}`
	channel.Setting = common.GetPointer(invalidTarget)
	require.ErrorContains(t, channel.ValidateSettings(), "target_path")

	invalidPath := `{"request_path_rewrite":[{"conditions":{"request_path":"/v1/unknown"},"target_path":"/custom"}]}`
	channel.Setting = common.GetPointer(invalidPath)
	require.ErrorContains(t, channel.ValidateSettings(), "conditions.request_path")
}
