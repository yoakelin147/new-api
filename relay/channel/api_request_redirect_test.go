package channel

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeepUpstreamRedirectResponse(t *testing.T) {
	assert.True(t, errors.Is(keepUpstreamRedirectResponse(nil, nil), http.ErrUseLastResponse))
}
