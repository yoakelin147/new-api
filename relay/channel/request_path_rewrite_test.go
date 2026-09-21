package channel

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestApplyRequestPathRewrite(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelSetting: dto.ChannelSettings{
		RequestPathRewrite: []dto.RequestPathRewriteRule{{
			TargetPath: "/apps/anthropic/v1/messages",
			Conditions: map[string]string{"request_path": "/v1/messages"},
		}},
	}}}

	got, err := applyRequestPathRewrite(ctx, info, "https://workspace.maas.aliyuncs.com/compatible-mode/v1/messages")

	require.NoError(t, err)
	require.Equal(t, "https://workspace.maas.aliyuncs.com/apps/anthropic/v1/messages", got)
}

func TestApplyRequestPathRewriteSupportsAllEntryPaths(t *testing.T) {
	for _, requestPath := range []string{"/v1/chat/completions", "/v1/responses", "/v1/messages"} {
		t.Run(requestPath, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("POST", requestPath, nil)
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
				ChannelSetting: dto.ChannelSettings{RequestPathRewrite: []dto.RequestPathRewriteRule{{
					TargetPath: "/custom/upstream",
					Conditions: map[string]string{"request_path": requestPath},
				}}},
			}}

			got, err := applyRequestPathRewrite(ctx, info, "https://provider.example/default")

			require.NoError(t, err)
			require.Equal(t, "https://provider.example/custom/upstream", got)
		})
	}
}

func TestApplyRequestPathRewriteKeepsDefaultURLWithoutConfiguration(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}

	got, err := applyRequestPathRewrite(ctx, info, "https://provider.example/v1/messages")

	require.NoError(t, err)
	require.Equal(t, "https://provider.example/v1/messages", got)
}

func TestApplyRequestPathRewriteMatchesConvertedRequestPath(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	info := &relaycommon.RelayInfo{
		FinalRequestRelayFormat: types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{ChannelSetting: dto.ChannelSettings{
			RequestPathRewrite: []dto.RequestPathRewriteRule{
				{
					TargetPath: "/wrong/messages-target",
					Conditions: map[string]string{"request_path": "/v1/messages"},
				},
				{
					TargetPath: "/custom/chat-target",
					Conditions: map[string]string{"request_path": "/v1/chat/completions"},
				},
			},
		}},
	}

	got, err := applyRequestPathRewrite(ctx, info, "https://provider.example/v1/chat/completions")

	require.NoError(t, err)
	require.Equal(t, "https://provider.example/custom/chat-target", got)
}

func TestApplyRequestPathRewriteLeavesUnmatchedPathUntouched(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelSetting: dto.ChannelSettings{
		RequestPathRewrite: []dto.RequestPathRewriteRule{{
			TargetPath: "/apps/anthropic/v1/messages",
			Conditions: map[string]string{"request_path": "/v1/messages"},
		}},
	}}}

	got, err := applyRequestPathRewrite(ctx, info, "https://provider.example/v1/responses")

	require.NoError(t, err)
	require.Equal(t, "https://provider.example/v1/responses", got)
}

func TestApplyRequestPathRewriteRejectsAbsoluteTarget(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelSetting: dto.ChannelSettings{RequestPathRewrite: []dto.RequestPathRewriteRule{{
			TargetPath: "https://attacker.example/v1/messages",
			Conditions: map[string]string{"request_path": "/v1/messages"},
		}}},
	}}

	_, err := applyRequestPathRewrite(ctx, info, "https://provider.example/v1/messages")

	require.ErrorContains(t, err, "must start with")
}
