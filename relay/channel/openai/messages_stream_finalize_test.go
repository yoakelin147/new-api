package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOaiStreamHandlerFinalizesConvertedMessagesWithoutUsageChunk(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	body := strings.Join([]string{
		`data: {"id":"chatcmpl-1","model":"kimi-k3","choices":[{"index":0,"delta":{"role":"assistant","reasoning_content":"thinking"},"finish_reason":null}]}`,
		``,
		`data: {"id":"chatcmpl-1","model":"kimi-k3","choices":[{"index":0,"delta":{"content":"answer"},"finish_reason":null}]}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}
	info := &relaycommon.RelayInfo{
		RelayFormat:             types.RelayFormatClaude,
		FinalRequestRelayFormat: types.RelayFormatOpenAI,
		RelayMode:               relayconstant.RelayModeUnknown,
		OriginModelName:         "kimi-k3",
		IsStream:                true,
		DisablePing:             true,
		ClaudeConvertInfo:       &relaycommon.ClaudeConvertInfo{LastMessagesType: relaycommon.LastMessageTypeNone},
		ChannelMeta:             &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeMoonshot, UpstreamModelName: "kimi-k3"},
	}

	usage, err := OaiStreamHandler(c, info, resp)

	require.Nil(t, err)
	require.Positive(t, usage.CompletionTokens)
	require.Contains(t, recorder.Body.String(), `"type":"content_block_stop"`)
	require.Contains(t, recorder.Body.String(), `"type":"message_delta"`)
	require.Contains(t, recorder.Body.String(), `"type":"message_stop"`)
	require.Contains(t, recorder.Body.String(), `"output_tokens":`)
	require.NotContains(t, recorder.Body.String(), `claude_cache_creation_5_m_tokens`)
	require.NotContains(t, recorder.Body.String(), `claude_cache_creation_1_h_tokens`)
}
