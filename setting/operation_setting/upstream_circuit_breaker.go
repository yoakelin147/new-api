package operation_setting

import (
	"time"

	"github.com/QuantumNous/new-api/setting/config"
)

type UpstreamCircuitBreakerSetting struct {
	Enabled                      bool `json:"enabled"`
	TimeoutSeconds               int  `json:"timeout_seconds"`
	RetryEnabled                 bool `json:"retry_enabled"`
	RetryBeforeFirstResponseOnly bool `json:"retry_before_first_response_only"`
}

var upstreamCircuitBreakerSetting = UpstreamCircuitBreakerSetting{
	Enabled:                      false,
	TimeoutSeconds:               300,
	RetryEnabled:                 true,
	RetryBeforeFirstResponseOnly: true,
}

func init() {
	config.GlobalConfig.Register("upstream_circuit_breaker", &upstreamCircuitBreakerSetting)
}

func GetUpstreamCircuitBreakerSetting() *UpstreamCircuitBreakerSetting {
	return &upstreamCircuitBreakerSetting
}

func IsUpstreamCircuitBreakerEnabled() bool {
	return upstreamCircuitBreakerSetting.Enabled && upstreamCircuitBreakerSetting.TimeoutSeconds > 0
}

func IsUpstreamCircuitBreakerRetryEnabled() bool {
	return IsUpstreamCircuitBreakerEnabled() && upstreamCircuitBreakerSetting.RetryEnabled
}

func UpstreamCircuitBreakerTimeout() time.Duration {
	if !IsUpstreamCircuitBreakerEnabled() {
		return 0
	}
	return time.Duration(upstreamCircuitBreakerSetting.TimeoutSeconds) * time.Second
}
