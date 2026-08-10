package common

import (
	"io"
	"testing"

	projectcommon "github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOutboundJSONBodyReplaysIndependentReaders(t *testing.T) {
	payload := []byte(`{"model":"test-model","messages":[{"role":"user","content":"hello"}]}`)

	body, closer, err := NewOutboundJSONBody(payload)
	require.NoError(t, err)
	defer closer.Close()

	first, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, payload, first)

	reader, err := body.NewReader()
	require.NoError(t, err)
	replay, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	assert.Equal(t, payload, replay)

	require.NoError(t, closer.Close())
	_, err = body.NewReader()
	assert.ErrorIs(t, err, projectcommon.ErrStorageClosed)
}
