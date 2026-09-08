package relay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// Exercise the actual outbound path; the mock returns a terminal 400 after
// recording the body, avoiding database/billing side effects in this test.
func TestKimiTextHelperOutboundIsolation(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	old := *settings
	settings.PassThroughRequestEnabled = false
	settings.ChatCompletionsToResponsesPolicy = model_setting.ChatCompletionsToResponsesPolicy{}
	t.Cleanup(func() { *settings = old })
	for _, tc := range []struct {
		name              string
		channelType       int
		model             string
		passthrough       bool
		globalPassthrough bool
		wantTools         bool
	}{
		{"moonshot", constant.ChannelTypeMoonshot, "kimi-k3", false, false, true},
		{"future", constant.ChannelTypeMoonshot, "kimi-k4", false, false, true},
		{"openai", constant.ChannelTypeOpenAI, "kimi-k3", false, false, false},
		{"k2", constant.ChannelTypeMoonshot, "kimi-k2.5", false, false, false},
		{"passthrough", constant.ChannelTypeMoonshot, "kimi-k3", true, false, true},
		{"global-passthrough", constant.ChannelTypeOpenAI, "kimi-k3", false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings.PassThroughRequestEnabled = tc.globalPassthrough
			captured := make(chan string, 1)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data, _ := io.ReadAll(r.Body)
				captured <- string(data)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(400)
				_, _ = io.WriteString(w, `{"error":{"message":"mock upstream boundary","type":"invalid_request_error"}}`)
			}))
			defer upstream.Close()
			for _, stream := range []string{"false", "true"} {
				body := `{"model":"public-alias","stream":` + stream + `,"messages":[{"role":"system","content":"","tools":[{"type":"function","function":{"name":"weather","strict":false,"parameters":{"type":"object"}}}]},{"role":"user","content":"weather?"}],"thinking":{"type":"enabled","effort":"max"},"tool_choice":"required","temperature":0,"service_tier":"priority"}`
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
				c.Request.Header.Set("Content-Type", "application/json")
				common.SetContextKey(c, constant.ContextKeyChannelType, tc.channelType)
				common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, upstream.URL)
				common.SetContextKey(c, constant.ContextKeyOriginalModel, "public-alias")
				common.SetContextKey(c, constant.ContextKeyChannelSetting, dto.ChannelSettings{SystemPrompt: "configured prompt", SystemPromptOverride: true, PassThroughBodyEnabled: tc.passthrough})
				common.SetContextKey(c, constant.ContextKeyChannelParamOverride, map[string]any{"temperature": 0.7})
				mapping, err := common.Marshal(map[string]string{"public-alias": tc.model})
				require.NoError(t, err)
				c.Set("model_mapping", string(mapping))
				request, err := helper.GetAndValidateTextRequest(c, relayconstant.RelayModeChatCompletions)
				require.NoError(t, err)
				info := &relaycommon.RelayInfo{Request: request, OriginModelName: "public-alias", RelayMode: relayconstant.RelayModeChatCompletions, DisablePing: true, IsStream: stream == "true"}
				relayErr := TextHelper(c, info)
				require.NotNil(t, relayErr)
				require.Equal(t, 400, relayErr.StatusCode)
				require.Contains(t, relayErr.Error(), "mock upstream boundary")
				got := <-captured
				if tc.passthrough || tc.globalPassthrough {
					require.JSONEq(t, body, got)
				} else {
					require.Equal(t, tc.model, gjson.Get(got, "model").String())
					require.Equal(t, 0.7, gjson.Get(got, "temperature").Float())
					require.False(t, gjson.Get(got, "service_tier").Exists())
					require.Equal(t, "max", gjson.Get(got, "thinking.effort").String())
					if tc.wantTools {
						require.Equal(t, "configured prompt", gjson.Get(got, "messages.0.content").String())
						require.Equal(t, "", gjson.Get(got, "messages.1.content").String())
						require.Equal(t, "weather", gjson.Get(got, "messages.1.tools.0.function.name").String())
						require.True(t, gjson.Get(got, "messages.1.tools.0.function.strict").Exists())
					} else {
						require.NotContains(t, got, `"tools"`)
					}
				}
				common.CleanupBodyStorage(c)
			}
		})
	}
}
