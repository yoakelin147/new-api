package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestShouldRetryNonStreamClientDisconnectUsesStatusCodeRules(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)

	originalRanges := operation_setting.AutomaticRetryStatusCodeRanges
	t.Cleanup(func() { operation_setting.AutomaticRetryStatusCodeRanges = originalRanges })
	operation_setting.AutomaticRetryStatusCodeRanges = []operation_setting.StatusCodeRange{
		{Start: 500, End: 503},
	}

	err := types.NewError(
		contextCanceledError{},
		types.ErrorCodeDoRequestFailed,
		types.ErrOptionWithStatusCode(500),
		types.ErrOptionWithHideErrMsg("client disconnected during upstream request"),
	)

	require.True(t, shouldRetry(ctx, err, 1))
}

type contextCanceledError struct{}

func (contextCanceledError) Error() string { return "context canceled" }
