package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateFaultHandlingOption(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{name: "minimum interval", key: "fault_handling.clickhouse_probe_interval_seconds", value: "5"},
		{name: "maximum interval", key: "fault_handling.clickhouse_probe_interval_seconds", value: "3600"},
		{name: "interval too low", key: "fault_handling.clickhouse_probe_interval_seconds", value: "4", wantErr: true},
		{name: "interval too high", key: "fault_handling.clickhouse_probe_interval_seconds", value: "3601", wantErr: true},
		{name: "minimum timeout", key: "fault_handling.clickhouse_probe_timeout_milliseconds", value: "100"},
		{name: "maximum timeout", key: "fault_handling.clickhouse_probe_timeout_milliseconds", value: "10000"},
		{name: "timeout too low", key: "fault_handling.clickhouse_probe_timeout_milliseconds", value: "99", wantErr: true},
		{name: "timeout too high", key: "fault_handling.clickhouse_probe_timeout_milliseconds", value: "10001", wantErr: true},
		{name: "valid enabled", key: "fault_handling.clickhouse_auto_recovery_enabled", value: "true"},
		{name: "invalid enabled", key: "fault_handling.clickhouse_auto_recovery_enabled", value: "yes", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateFaultHandlingOption(test.key, test.value)
			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
