package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateWebhookURL(t *testing.T) {
	require.NoError(t, ValidateWebhookURL(""))
	require.NoError(t, ValidateWebhookURL("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test"))
	require.Error(t, ValidateWebhookURL("http://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test"))
	require.Error(t, ValidateWebhookURL("https://example.com/cgi-bin/webhook/send?key=test"))
	require.Error(t, ValidateWebhookURL("https://qyapi.weixin.qq.com/cgi-bin/webhook/send"))
}
