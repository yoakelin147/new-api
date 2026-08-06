package openai

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRealtimeHeadersOnlyUseBetaForPreviewModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name      string
		model     string
		websocket bool
		wantBeta  string
		wantProto string
	}{
		{name: "ga http", model: "gpt-realtime", wantBeta: ""},
		{name: "preview http", model: "gpt-4o-realtime-preview", wantBeta: "realtime=v1"},
		{name: "ga websocket", model: "gpt-realtime", websocket: true, wantProto: "realtime,openai-insecure-api-key.secret"},
		{name: "preview websocket", model: "gpt-4o-realtime-preview", websocket: true, wantProto: "realtime,openai-insecure-api-key.secret,openai-beta.realtime-v1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v1/realtime", nil)
			if test.websocket {
				request.Header.Set("Sec-WebSocket-Protocol", "realtime")
			}
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Request = request
			headers := http.Header{}
			err := (&Adaptor{}).SetupRequestHeader(context, &headers, &relaycommon.RelayInfo{
				RelayMode: relayconstant.RelayModeRealtime,
				ChannelMeta: &relaycommon.ChannelMeta{
					ApiKey: "secret", ChannelType: constant.ChannelTypeOpenAI,
					UpstreamModelName: test.model,
				},
			})
			require.NoError(t, err)
			require.Equal(t, test.wantBeta, headers.Get("openai-beta"))
			require.Equal(t, test.wantProto, headers.Get("Sec-WebSocket-Protocol"))
		})
	}
}
