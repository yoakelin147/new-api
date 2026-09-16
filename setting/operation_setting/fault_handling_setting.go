package operation_setting

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/QuantumNous/new-api/setting/config"
)

const (
	FaultHandlingProbeIntervalMinSeconds = 5
	FaultHandlingProbeIntervalMaxSeconds = 3600
	FaultHandlingProbeTimeoutMinMillis   = 100
	FaultHandlingProbeTimeoutMaxMillis   = 10000
)

type FaultHandlingSetting struct {
	ClickHouseAutoRecoveryEnabled  bool   `json:"clickhouse_auto_recovery_enabled"`
	ClickHouseProbeIntervalSeconds int    `json:"clickhouse_probe_interval_seconds"`
	ClickHouseProbeTimeoutMillis   int    `json:"clickhouse_probe_timeout_milliseconds"`
	ClickHouseAlertEnabled         bool   `json:"clickhouse_alert_enabled"`
	ClickHouseAlertWebhookURL      string `json:"clickhouse_alert_webhook_url"`
}

var (
	faultHandlingSetting = FaultHandlingSetting{
		ClickHouseAutoRecoveryEnabled:  true,
		ClickHouseProbeIntervalSeconds: 30,
		ClickHouseProbeTimeoutMillis:   1000,
		ClickHouseAlertEnabled:         false,
	}
	faultHandlingAutoRecovery atomic.Bool
	faultHandlingInterval     atomic.Int64
	faultHandlingTimeout      atomic.Int64
	faultHandlingAlertEnabled atomic.Bool
	faultHandlingWebhookURL   atomic.Value
)

func init() {
	config.GlobalConfig.Register("fault_handling", &faultHandlingSetting)
	UpdateFaultHandlingSetting()
}

func UpdateFaultHandlingSetting() {
	faultHandlingSetting.ClickHouseProbeIntervalSeconds = min(
		max(faultHandlingSetting.ClickHouseProbeIntervalSeconds, FaultHandlingProbeIntervalMinSeconds),
		FaultHandlingProbeIntervalMaxSeconds,
	)
	faultHandlingSetting.ClickHouseProbeTimeoutMillis = min(
		max(faultHandlingSetting.ClickHouseProbeTimeoutMillis, FaultHandlingProbeTimeoutMinMillis),
		FaultHandlingProbeTimeoutMaxMillis,
	)
	faultHandlingAutoRecovery.Store(faultHandlingSetting.ClickHouseAutoRecoveryEnabled)
	faultHandlingInterval.Store(int64(faultHandlingSetting.ClickHouseProbeIntervalSeconds))
	faultHandlingTimeout.Store(int64(faultHandlingSetting.ClickHouseProbeTimeoutMillis))
	faultHandlingAlertEnabled.Store(faultHandlingSetting.ClickHouseAlertEnabled)
	if ValidateWebhookURL(faultHandlingSetting.ClickHouseAlertWebhookURL) == nil {
		faultHandlingWebhookURL.Store(strings.TrimSpace(faultHandlingSetting.ClickHouseAlertWebhookURL))
	} else {
		faultHandlingWebhookURL.Store("")
	}
}

func GetClickHouseAlertConfig() (enabled bool, webhookURL string) {
	webhookURL, _ = faultHandlingWebhookURL.Load().(string)
	return faultHandlingAlertEnabled.Load(), webhookURL
}

func IsClickHouseAlertWebhookConfigured() bool {
	webhookURL, _ := faultHandlingWebhookURL.Load().(string)
	return webhookURL != ""
}

func ValidateWebhookURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Scheme != "https" || u.Host != "qyapi.weixin.qq.com" || u.Path != "/cgi-bin/webhook/send" || u.User != nil || u.Fragment != "" || u.Query().Get("key") == "" {
		return fmt.Errorf("企业微信 Webhook 必须是 https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=... 格式")
	}
	return nil
}

func GetFaultHandlingSetting() FaultHandlingSetting {
	webhookURL, _ := faultHandlingWebhookURL.Load().(string)
	return FaultHandlingSetting{
		ClickHouseAutoRecoveryEnabled:  faultHandlingAutoRecovery.Load(),
		ClickHouseProbeIntervalSeconds: int(faultHandlingInterval.Load()),
		ClickHouseProbeTimeoutMillis:   int(faultHandlingTimeout.Load()),
		ClickHouseAlertEnabled:         faultHandlingAlertEnabled.Load(),
		ClickHouseAlertWebhookURL:      webhookURL,
	}
}

func ValidateFaultHandlingOption(key string, value string) error {
	value = strings.TrimSpace(value)
	switch key {
	case "fault_handling.clickhouse_auto_recovery_enabled":
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("ClickHouse 自动恢复探测开关必须是布尔值")
		}
	case "fault_handling.clickhouse_probe_interval_seconds":
		return validateFaultHandlingRange(value, FaultHandlingProbeIntervalMinSeconds, FaultHandlingProbeIntervalMaxSeconds, "ClickHouse 探测间隔")
	case "fault_handling.clickhouse_probe_timeout_milliseconds":
		return validateFaultHandlingRange(value, FaultHandlingProbeTimeoutMinMillis, FaultHandlingProbeTimeoutMaxMillis, "ClickHouse 探测超时")
	case "fault_handling.clickhouse_alert_enabled":
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("ClickHouse 故障告警开关必须是布尔值")
		}
	case "fault_handling.clickhouse_alert_webhook_url":
		return ValidateWebhookURL(value)
	}
	return nil
}

func validateFaultHandlingRange(value string, minValue int, maxValue int, name string) error {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minValue || parsed > maxValue {
		return fmt.Errorf("%s必须是 %d-%d 之间的整数", name, minValue, maxValue)
	}
	return nil
}
