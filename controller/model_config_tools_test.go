package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/require"
)

func TestNormalizeAIEndpointMetadataConvertsPathArrayToEndpointObject(t *testing.T) {
	got := normalizeAIEndpointMetadata([]byte(`["/v1/chat/completions","/v1/embeddings"]`))

	var endpoints map[string]common.EndpointInfo
	require.NoError(t, common.Unmarshal(got, &endpoints))
	require.Equal(t, map[string]common.EndpointInfo{
		"embeddings": {
			Path:   "/v1/embeddings",
			Method: "POST",
		},
		"openai": {
			Path:   "/v1/chat/completions",
			Method: "POST",
		},
	}, endpoints)
}

func TestNormalizeAIEndpointMetadataConvertsOpenAIVideoPath(t *testing.T) {
	got := normalizeAIEndpointMetadata([]byte(`["/v1/videos"]`))

	var endpoints map[string]common.EndpointInfo
	require.NoError(t, common.Unmarshal(got, &endpoints))
	require.Equal(t, map[string]common.EndpointInfo{
		"openai-video": {
			Path:   "/v1/videos",
			Method: "POST",
		},
	}, endpoints)
}

func TestNormalizeAIEndpointMetadataConvertsAsyncImagePath(t *testing.T) {
	got := normalizeAIEndpointMetadata([]byte(`["/v1/images/generations/async"]`))

	var endpoints map[string]common.EndpointInfo
	require.NoError(t, common.Unmarshal(got, &endpoints))
	require.Equal(t, map[string]common.EndpointInfo{
		"image-generation-async": {
			Path:   "/v1/images/generations/async",
			Method: "POST",
		},
	}, endpoints)
}

func TestNormalizeAIEndpointMetadataConvertsLegacyVideoGenerationPath(t *testing.T) {
	got := normalizeAIEndpointMetadata([]byte(`["/v1/video/generations"]`))

	var endpoints map[string]common.EndpointInfo
	require.NoError(t, common.Unmarshal(got, &endpoints))
	require.Equal(t, map[string]common.EndpointInfo{
		"openai-video": {
			Path:   "/v1/videos",
			Method: "POST",
		},
	}, endpoints)
}

func TestNormalizeAIEndpointMetadataDropsUnknownPaths(t *testing.T) {
	got := normalizeAIEndpointMetadata([]byte(`["/unknown"]`))

	require.Nil(t, got)
}

func TestNormalizeAIEndpointMetadataConvertsPathObjectKeys(t *testing.T) {
	got := normalizeAIEndpointMetadata([]byte(`{"/v1/chat/completions":{"path":"/v1/chat/completions","method":"POST"}}`))

	var endpoints map[string]common.EndpointInfo
	require.NoError(t, common.Unmarshal(got, &endpoints))
	require.Equal(t, map[string]common.EndpointInfo{
		"openai": {
			Path:   "/v1/chat/completions",
			Method: "POST",
		},
	}, endpoints)
}

func TestNormalizeUpstreamEndpointsConvertsAnthropicArray(t *testing.T) {
	got := normalizeUpstreamEndpoints([]byte(`["anthropic"]`))

	var endpoints map[string]common.EndpointInfo
	require.NoError(t, common.Unmarshal([]byte(got), &endpoints))
	require.Equal(t, map[string]common.EndpointInfo{
		"anthropic": {
			Path:   "/v1/messages",
			Method: "POST",
		},
	}, endpoints)
}

func TestNormalizeModelEndpointsDetectsOpenAIAnthropicDifference(t *testing.T) {
	local := normalizeModelEndpoints(`["openai"]`)
	upstream := normalizeUpstreamEndpoints([]byte(`{"anthropic":{"path":"/v1/messages","method":"POST"}}`))

	require.NotEmpty(t, local)
	require.NotEmpty(t, upstream)
	require.NotEqual(t, local, upstream)
}
