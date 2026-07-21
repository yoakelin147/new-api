package common

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/tidwall/gjson"
)

const (
	mappedTaskStatusQueued     = "QUEUED"
	mappedTaskStatusInProgress = "IN_PROGRESS"
	mappedTaskStatusSuccess    = "SUCCESS"
	mappedTaskStatusFailure    = "FAILURE"
)

func ExtractAsyncTaskMappedSubmitID(respBody []byte, mapping *dto.AsyncTaskResponseMapping) string {
	if mapping == nil {
		return ""
	}
	return ExtractAsyncTaskMappedString(respBody, mapping.SubmitTaskIDPaths)
}

func ParseAsyncTaskResultWithMapping(respBody []byte, mapping *dto.AsyncTaskResponseMapping) (*TaskInfo, bool) {
	if mapping == nil {
		return nil, false
	}
	statusRaw := ExtractAsyncTaskMappedString(respBody, mapping.FetchStatusPaths)
	status := normalizeAsyncTaskMappedStatus(statusRaw, mapping.FetchStatusMap)
	if status == "" {
		return nil, false
	}

	taskResult := &TaskInfo{
		Code:   0,
		Status: status,
	}
	if progress := normalizeAsyncTaskMappedProgress(ExtractAsyncTaskMappedString(respBody, mapping.FetchProgressPaths)); progress != "" {
		taskResult.Progress = progress
	} else if status == mappedTaskStatusSuccess {
		taskResult.Progress = "100%"
	}
	if status == mappedTaskStatusSuccess {
		taskResult.Url = ExtractAsyncTaskMappedString(respBody, mapping.FetchResultURLPaths)
	}
	if status == mappedTaskStatusFailure {
		if reason := ExtractAsyncTaskMappedString(respBody, mapping.FetchErrorMessagePaths); reason != "" {
			taskResult.Reason = reason
		} else {
			taskResult.Reason = "task failed"
		}
	}
	return taskResult, true
}

func ExtractAsyncTaskMappedString(respBody []byte, paths []string) string {
	for _, path := range paths {
		value := extractAsyncTaskMappedValue(respBody, path)
		if value != "" {
			return value
		}
	}
	return ""
}

func extractAsyncTaskMappedValue(respBody []byte, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	value := gjson.GetBytes(respBody, path)
	if !value.Exists() {
		return ""
	}
	switch value.Type {
	case gjson.String:
		return strings.TrimSpace(value.String())
	case gjson.Number, gjson.True, gjson.False:
		return strings.TrimSpace(value.String())
	case gjson.JSON:
		if value.IsArray() {
			for _, item := range value.Array() {
				if out := strings.TrimSpace(item.String()); out != "" {
					return out
				}
			}
		}
	}
	return ""
}

func normalizeAsyncTaskMappedStatus(status string, statusMap map[string]string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return ""
	}
	for key, value := range statusMap {
		if key == status || strings.EqualFold(key, status) {
			if normalized := normalizeAsyncTaskStatusValue(value); normalized != "" {
				return normalized
			}
		}
	}
	return normalizeAsyncTaskStatusValue(status)
}

func normalizeAsyncTaskStatusValue(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "queued", "queueing", "pending", "submitted":
		return mappedTaskStatusQueued
	case "processing", "running", "in_progress", "in-progress":
		return mappedTaskStatusInProgress
	case "completed", "succeeded", "success", "done", "finished":
		return mappedTaskStatusSuccess
	case "failed", "failure", "cancelled", "canceled", "error":
		return mappedTaskStatusFailure
	default:
		return ""
	}
}

func normalizeAsyncTaskMappedProgress(progress string) string {
	progress = strings.TrimSpace(progress)
	if progress == "" {
		return ""
	}
	if strings.HasSuffix(progress, "%") {
		return progress
	}
	value, err := strconv.ParseFloat(progress, 64)
	if err != nil || value < 0 || value > 100 {
		return progress
	}
	if value == float64(int(value)) {
		return fmt.Sprintf("%d%%", int(value))
	}
	return fmt.Sprintf("%.2f%%", value)
}
