package helper

import (
	"fmt"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func UpstreamCircuitBreakerStreamError(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	if info == nil || info.StreamStatus == nil {
		return nil
	}
	if info.StreamStatus.EndReason != relaycommon.StreamEndReasonTimeout {
		return nil
	}
	setting := operation_setting.GetUpstreamCircuitBreakerSetting()
	if !operation_setting.IsUpstreamCircuitBreakerRetryEnabled() {
		return nil
	}
	if setting.RetryBeforeFirstResponseOnly && info.ReceivedResponseCount > 0 {
		return nil
	}
	if c != nil && c.Request != nil && c.Request.Context().Err() != nil {
		return nil
	}
	return types.NewError(
		fmt.Errorf("upstream circuit breaker stream timeout before first response after %d seconds", setting.TimeoutSeconds),
		types.ErrorCodeChannelUpstreamTimeout,
		types.ErrOptionWithStatusCode(599),
	)
}
