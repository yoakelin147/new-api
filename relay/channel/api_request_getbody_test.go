package channel

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	projectcommon "github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyUpstreamBodyMetadataSetsReplayableMetadata(t *testing.T) {
	payload := []byte(`{"model":"test-model"}`)
	body, closer, err := relaycommon.NewOutboundJSONBody(payload)
	require.NoError(t, err)
	defer closer.Close()

	req, err := http.NewRequest(http.MethodPost, "https://example.com", body)
	require.NoError(t, err)
	assert.Zero(t, req.ContentLength)
	assert.Nil(t, req.GetBody)

	ApplyUpstreamBodyMetadata(req, body)

	assert.EqualValues(t, len(payload), req.ContentLength)
	require.NotNil(t, req.GetBody)
	_, err = io.ReadAll(req.Body)
	require.NoError(t, err)
	replayBody, err := req.GetBody()
	require.NoError(t, err)
	replay, err := io.ReadAll(replayBody)
	require.NoError(t, err)
	require.NoError(t, replayBody.Close())
	assert.Equal(t, payload, replay)
}

func TestApplyUpstreamBodyMetadataPreservesNativeGetBody(t *testing.T) {
	body := bytes.NewReader([]byte("original"))
	req, err := http.NewRequest(http.MethodPost, "https://example.com", body)
	require.NoError(t, err)
	require.NotNil(t, req.GetBody)

	ApplyUpstreamBodyMetadata(req, body)

	replayBody, err := req.GetBody()
	require.NoError(t, err)
	replay, err := io.ReadAll(replayBody)
	require.NoError(t, err)
	require.NoError(t, replayBody.Close())
	assert.Equal(t, []byte("original"), replay)
	assert.EqualValues(t, len("original"), req.ContentLength)
}

func TestApplyUpstreamBodyMetadataHidesStorageCloser(t *testing.T) {
	payload := []byte(`{"model":"test-model"}`)
	storage, err := projectcommon.CreateBodyStorage(payload)
	require.NoError(t, err)
	defer storage.Close()

	req, err := http.NewRequest(http.MethodPost, "https://example.com", storage)
	require.NoError(t, err)
	ApplyUpstreamBodyMetadata(req, storage)

	_, exposesStorage := req.Body.(projectcommon.BodyStorage)
	assert.False(t, exposesStorage)
	assert.EqualValues(t, len(payload), req.ContentLength)
	require.NotNil(t, req.GetBody)
	require.NoError(t, req.Body.Close())
	replayBody, err := req.GetBody()
	require.NoError(t, err)
	replay, err := io.ReadAll(replayBody)
	require.NoError(t, err)
	require.NoError(t, replayBody.Close())
	assert.Equal(t, payload, replay)
}
