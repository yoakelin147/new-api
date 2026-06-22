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
