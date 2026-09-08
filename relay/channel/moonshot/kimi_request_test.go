package moonshot

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

const dynamicTool = `{"type":"function","function":{"name":"get_weather","strict":false,"parameters":{"type":"object","properties":{"city":{"type":"string"}}}},"future_option":false}`

func kimiTestRequest(t *testing.T, body string) (*gin.Context, *dto.GeneralOpenAIRequest) {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	t.Cleanup(func() { common.CleanupBodyStorage(c) })
	var request dto.GeneralOpenAIRequest
	require.NoError(t, common.UnmarshalBodyReusable(c, &request))
	// The relay copies the parsed DTO for each channel attempt.
	copied, err := common.DeepCopy(&request)
	require.NoError(t, err)
	return c, copied
}

func TestKimiDynamicToolsPreservedWithMappedModel(t *testing.T) {
	for _, stream := range []string{"false", "true"} {
		t.Run(stream, func(t *testing.T) {
			body := `{"model":"customer-alias","stream":` + stream + `,"messages":[{"role":"system","content":"","tools":[` + dynamicTool + `]},{"role":"user","content":"weather?"}],"tool_choice":"required","thinking":{"type":"enabled","keep":"all","effort":"max","future_option":0},"reasoning_effort":"low"}`
			c, request := kimiTestRequest(t, body)
			request.Model = "kimi-k3"
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeMoonshot, UpstreamModelName: "kimi-k3"}}
			converted, err := (&Adaptor{}).ConvertOpenAIRequest(c, info, request)
			require.NoError(t, err)
			encoded, err := common.Marshal(converted)
			require.NoError(t, err)
			require.JSONEq(t, strings.Replace(body, "customer-alias", "kimi-k3", 1), string(encoded))
			// Conversion must not change the stored body used by later retries.
			storage, err := common.GetBodyStorage(c)
			require.NoError(t, err)
			original, err := storage.Bytes()
			require.NoError(t, err)
			require.JSONEq(t, body, string(original))
		})
	}
}

func TestKimiDynamicToolsRejectedAsBadRequest(t *testing.T) {
	tests := map[string]string{
		"user role":          `{"role":"user","content":"","tools":[` + dynamicTool + `]}`,
		"assistant role":     `{"role":"assistant","content":"","tools":[]}`,
		"content":            `{"role":"system","content":"nonempty","tools":[` + dynamicTool + `]}`,
		"not array":          `{"role":"system","content":"","tools":{}}`,
		"null array":         `{"role":"system","content":"","tools":null}`,
		"null item":          `{"role":"system","content":"","tools":[null]}`,
		"scalar item":        `{"role":"system","content":"","tools":[42]}`,
		"missing type":       `{"role":"system","content":"","tools":[{"function":{"name":"valid"}}]}`,
		"bogus type":         `{"role":"system","content":"","tools":[{"type":"bogus","function":{"name":"valid"}}]}`,
		"missing function":   `{"role":"system","content":"","tools":[{"type":"function"}]}`,
		"missing name":       `{"role":"system","content":"","tools":[{"type":"function","function":{}}]}`,
		"numeric name":       `{"role":"system","content":"","tools":[{"type":"function","function":{"name":123}}]}`,
		"empty name":         `{"role":"system","content":"","tools":[{"type":"function","function":{"name":""}}]}`,
		"starts with number": `{"role":"system","content":"","tools":[{"type":"function","function":{"name":"1bad"}}]}`,
		"special character":  `{"role":"system","content":"","tools":[{"type":"function","function":{"name":"bad@name"}}]}`,
		"too long":           `{"role":"system","content":"","tools":[{"type":"function","function":{"name":"` + strings.Repeat("a", 257) + `"}}]}`,
		"duplicate":          `{"role":"system","content":"","tools":[` + dynamicTool + `,` + dynamicTool + `]}`,
		"duplicate messages": `{"role":"system","content":"","tools":[` + dynamicTool + `]},{"role":"system","content":"","tools":[` + dynamicTool + `]}`,
	}
	for name, messages := range tests {
		t.Run(name, func(t *testing.T) {
			c, request := kimiTestRequest(t, `{"model":"kimi-k3","messages":[`+messages+`]}`)
			_, err := (&Adaptor{}).ConvertOpenAIRequest(c, nil, request)
			require.Error(t, err)
			// Same wrapping used by TextHelper must preserve HTTP 400 and skipRetry.
			relayErr := types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			require.Equal(t, http.StatusBadRequest, relayErr.StatusCode)
			require.True(t, types.IsSkipRetryError(relayErr))
		})
	}
}

func TestKimiOtherChannelsKeepExistingSerialization(t *testing.T) {
	body := `{"model":"kimi-k3","messages":[{"role":"system","content":"","tools":[` + dynamicTool + `]}]}`
	c, request := kimiTestRequest(t, body)
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, UpstreamModelName: "kimi-k3"}}
	converted, err := (&openai.Adaptor{}).ConvertOpenAIRequest(c, info, request)
	require.NoError(t, err)
	encoded, err := common.Marshal(converted)
	require.NoError(t, err)
	require.JSONEq(t, `{"model":"kimi-k3","messages":[{"role":"system","content":""}]}`, string(encoded))
	// The mapped upstream model, not the public alias, controls K3 compatibility.
	info.UpstreamModelName = "kimi-k2.5"
	converted, err = (&Adaptor{}).ConvertOpenAIRequest(c, info, request)
	require.NoError(t, err)
	encoded, err = common.Marshal(converted)
	require.NoError(t, err)
	require.JSONEq(t, `{"model":"kimi-k3","messages":[{"role":"system","content":""}]}`, string(encoded))
}

func TestKimiSeriesModelNames(t *testing.T) {
	require.Contains(t, ModelList, "kimi-k3")
	for _, model := range []string{"kimi-k3", "kimi-k3.5", "kimi-k3-preview", "kimi-k4", "kimi-k10", "moonshotai/Kimi-K3"} {
		require.True(t, isKimiK3OrLater(model), model)
	}
	for _, model := range []string{"kimi-k2.6", "gpt-5", "kimi-k", "kimi-k3fake", "kimi-k3/other", "kimi-k99999999999999999999999999"} {
		require.False(t, isKimiK3OrLater(model), model)
	}
}

func TestKimiGlobalAndDynamicToolNames(t *testing.T) {
	for _, globalName := range []string{"get_weather", "get_stock_price"} {
		c, request := kimiTestRequest(t, `{"model":"kimi-k4","tools":[{"type":"function","function":{"name":"`+globalName+`"}}],"messages":[{"role":"system","content":"","tools":[`+dynamicTool+`]}]}`)
		_, err := (&Adaptor{}).ConvertOpenAIRequest(c, nil, request)
		if globalName == "get_weather" {
			require.ErrorContains(t, err, "duplicate")
		} else {
			require.NoError(t, err)
		}
	}
}

func TestKimiThinkingNormalization(t *testing.T) {
	tests := []struct {
		name  string
		extra string
		want  string
	}{
		{name: "omitted"},
		{name: "enabled", extra: `,"thinking":{"type":"enabled","keep":"all"}`, want: "max"},
		{name: "explicit", extra: `,"thinking":{"type":"enabled","effort":"high"}`, want: "high"},
		{name: "reasoning effort", extra: `,"thinking":{"type":"enabled"},"reasoning_effort":"low"`},
		{name: "disabled", extra: `,"thinking":{"type":"disabled"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := `{"model":"kimi-k3","messages":[{"role":"user","content":"hi"}]` + test.extra + `}`
			c, request := kimiTestRequest(t, body)
			converted, err := (&Adaptor{}).ConvertOpenAIRequest(c, nil, request)
			require.NoError(t, err)
			encoded, err := common.Marshal(converted)
			require.NoError(t, err)
			if test.want == "" {
				require.False(t, gjson.GetBytes(encoded, "thinking.effort").Exists())
			} else {
				require.Equal(t, test.want, gjson.GetBytes(encoded, "thinking.effort").String())
			}
		})
	}
}
