package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/require"
)

// TestFormatUserLogsStripsQuotaSaturation verifies the admin-only quota
// saturation marker (nested under other.admin_info) is removed for non-admin
// log views, since formatUserLogs strips the whole admin_info object.
func TestFormatUserLogsStripsQuotaSaturation(t *testing.T) {
	other := common.MapToJsonStr(map[string]interface{}{
		"model_price": 0.004,
		"admin_info": map[string]interface{}{
			"quota_saturation": map[string]interface{}{
				"op":      "QuotaFromDecimal",
				"kind":    "overflow",
				"clamped": common.MaxQuota,
			},
		},
	})
	logs := []*Log{{Other: other}}

	formatUserLogs(logs, 0)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	_, hasAdminInfo := parsed["admin_info"]
	require.False(t, hasAdminInfo, "admin_info (and nested quota_saturation) must be stripped for non-admin views")
	// Non-admin billing fields remain visible.
	require.Contains(t, parsed, "model_price")
}

func TestFormatUserLogsStripsModelMappingDetails(t *testing.T) {
	other := common.MapToJsonStr(map[string]interface{}{
		"is_model_mapped":     true,
		"upstream_model_name": "GLM-5.2_Gem",
		"model_ratio":         0.45,
	})
	logs := []*Log{{
		ModelName: "glm-5.2",
		Other:     other,
	}}

	formatUserLogs(logs, 0)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	require.NotContains(t, parsed, "is_model_mapped")
	require.NotContains(t, parsed, "upstream_model_name")
	require.Equal(t, "glm-5.2", logs[0].ModelName)
	require.Contains(t, parsed, "model_ratio")
}

func TestFormatUserLogsStripsClientGoneBillingDetails(t *testing.T) {
	const internalMessage = "客户端已断开且未获取到上游输出 Token，本次请求不计费"
	other := common.MapToJsonStr(map[string]interface{}{
		"billing_waived_reason": "client_gone_without_upstream_output_tokens",
		"model_ratio":           1,
	})
	logs := []*Log{{
		Content: "模型 gpt-test, " + internalMessage,
		Other:   other,
	}}

	formatUserLogs(logs, 0)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	require.NotContains(t, parsed, "billing_waived_reason")
	require.Equal(t, "模型 gpt-test", logs[0].Content)
}
