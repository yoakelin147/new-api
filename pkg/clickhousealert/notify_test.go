package clickhousealert

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestBuildContentIncludesInstanceIdentity(t *testing.T) {
	t.Setenv("NODE_NAME", "sh-prod-api-02")
	t.Setenv("NODE_HOST_IP", "10.10.20.32")
	t.Setenv("CLICKHOUSE_NODE_ID", "7")
	oldVersion := common.Version
	common.Version = "test-version"
	t.Cleanup(func() { common.Version = oldVersion })

	content := buildContent(EventReadDegraded, "ClickHouse 读取失败")
	require.Contains(t, content, "sh-prod-api-02")
	require.Contains(t, content, "10.10.20.32")
	require.Contains(t, content, "日志节点 ID：7")
	require.Contains(t, content, "test-version")
	require.NotContains(t, content, "webhook")
}

func TestTruncateUTF8KeepsValidMarkdown(t *testing.T) {
	content := truncateUTF8(strings.Repeat("告警", 3000), 4096)
	require.LessOrEqual(t, len(content), 4096)
	require.True(t, strings.HasSuffix(content, "警") || strings.HasSuffix(content, "告"))
}
