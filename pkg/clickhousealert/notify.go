package clickhousealert

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

type Event string

const (
	EventWriteDegraded  Event = "write_degraded"
	EventWriteRecovered Event = "write_recovered"
	EventReadDegraded   Event = "read_degraded"
	EventReadRecovered  Event = "read_recovered"
)

type webhookPayload struct {
	MsgType  string `json:"msgtype"`
	Markdown struct {
		Content string `json:"content"`
	} `json:"markdown"`
}

type webhookResponse struct {
	ErrCode int `json:"errcode"`
}

func Notify(event Event, reason string) {
	enabled, webhookURL := operation_setting.GetClickHouseAlertConfig()
	if !enabled || webhookURL == "" {
		return
	}
	go send(webhookURL, event, reason)
}

func SendTest() error {
	_, webhookURL := operation_setting.GetClickHouseAlertConfig()
	if webhookURL == "" {
		return fmt.Errorf("企业微信 Webhook 尚未配置")
	}
	return sendRequest(webhookURL, buildContent(EventReadRecovered, "管理员测试消息"))
}

func send(webhookURL string, event Event, reason string) {
	if err := sendRequest(webhookURL, buildContent(event, reason)); err != nil {
		common.SysError("ClickHouse alert webhook returned an error")
	}
}

func sendRequest(webhookURL string, content string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	payload := webhookPayload{MsgType: "markdown"}
	payload.Markdown.Content = content
	body, err := common.Marshal(payload)
	if err != nil {
		return fmt.Errorf("payload encoding failed: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	var result webhookResponse
	if err := common.DecodeJson(io.LimitReader(resp.Body, 64*1024), &result); err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 || result.ErrCode != 0 {
		return fmt.Errorf("webhook returned status %d errcode %d", resp.StatusCode, result.ErrCode)
	}
	return nil
}

func buildContent(event Event, reason string) string {
	status, source := eventText(event)
	hostname, _ := os.Hostname()
	nodeName := strings.TrimSpace(os.Getenv("NODE_NAME"))
	if nodeName == "" {
		nodeName = "未配置"
	}
	hostIP := strings.TrimSpace(os.Getenv("NODE_HOST_IP"))
	if hostIP == "" {
		hostIP = "未配置"
	}
	nodeID := strings.TrimSpace(os.Getenv("CLICKHOUSE_NODE_ID"))
	if nodeID == "" {
		nodeID = "0"
	}
	content := fmt.Sprintf("# New API ClickHouse 故障告警\n> 状态：<font color=\"%s\">%s</font>\n> 节点名称：%s\n> 实例名称：%s\n> 宿主机 IP：%s\n> 实例内网 IP：%s\n> 日志节点 ID：%s\n> 应用版本：%s\n> 时间：%s\n> 当前%s：%s\n> 原因：%s", status.color, status.text, nodeName, emptyFallback(hostname), hostIP, emptyFallback(common.GetIp()), nodeID, common.Version, time.Now().Format("2006-01-02 15:04:05"), source.name, source.value, reason)
	return truncateUTF8(content, 4096)
}

type eventLabel struct {
	text, color string
}

type sourceLabel struct {
	name, value string
}

func eventText(event Event) (eventLabel, sourceLabel) {
	switch event {
	case EventWriteDegraded:
		return eventLabel{"日志写入已切换至备用数据库", "warning"}, sourceLabel{"写入源", "原日志数据库"}
	case EventWriteRecovered:
		return eventLabel{"日志写入已恢复至 ClickHouse", "info"}, sourceLabel{"写入源", "ClickHouse"}
	case EventReadDegraded:
		return eventLabel{"日志读取已切换至备用数据库", "warning"}, sourceLabel{"读取源", "原日志数据库"}
	default:
		return eventLabel{"日志读取已恢复至 ClickHouse", "info"}, sourceLabel{"读取源", "ClickHouse"}
	}
}

func emptyFallback(value string) string {
	if strings.TrimSpace(value) == "" {
		return "未识别"
	}
	return value
}

func truncateUTF8(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	value = value[:maxBytes]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}
