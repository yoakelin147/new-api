package relay

import (
	"encoding/json"
	"strings"
	"unicode"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
)

var publicTaskSensitiveKeys = map[string]struct{}{
	"accesstoken":       {},
	"apikey":            {},
	"authorization":     {},
	"billingcontext":    {},
	"channelid":         {},
	"credential":        {},
	"credentials":       {},
	"group":             {},
	"headers":           {},
	"key":               {},
	"platform":          {},
	"privatedata":       {},
	"project":           {},
	"projectid":         {},
	"quota":             {},
	"requestid":         {},
	"requestheaders":    {},
	"refreshtoken":      {},
	"secret":            {},
	"secretkey":         {},
	"token":             {},
	"upstreammodelname": {},
	"upstreamtaskid":    {},
	"userid":            {},
}

// TaskModel2PublicDto 将数据库任务转换为客户可见结构，不修改轮询、计费、
// 适配器和管理接口继续使用的原始任务记录。
func TaskModel2PublicDto(task *model.Task) *dto.PublicTaskDto {
	return &dto.PublicTaskDto{
		CreatedAt:  task.CreatedAt,
		UpdatedAt:  task.UpdatedAt,
		TaskID:     task.TaskID,
		Action:     task.Action,
		Status:     string(task.Status),
		FailReason: task.FailReason,
		ResultURL:  task.GetResultURL(),
		SubmitTime: task.SubmitTime,
		StartTime:  task.StartTime,
		FinishTime: task.FinishTime,
		Progress:   task.Progress,
		Properties: dto.PublicTaskProperties{
			Input:           task.Properties.Input,
			OriginModelName: task.Properties.OriginModelName,
		},
		Data: sanitizePublicTaskData(task),
	}
}

// TaskModel2PublicImageDto 只返回轮询异步图片任务及获取结果所需的字段。
func TaskModel2PublicImageDto(task *model.Task) *dto.PublicImageTaskDto {
	resultURL := task.PrivateData.ResultURL
	failReason := task.FailReason
	switch task.Status {
	case model.TaskStatusSuccess:
		if resultURL == "" {
			resultURL = task.FailReason
		}
		failReason = ""
	case model.TaskStatusFailure:
		resultURL = ""
	}

	return &dto.PublicImageTaskDto{
		TaskID:      task.TaskID,
		Status:      string(task.Status),
		Progress:    task.Progress,
		Model:       task.Properties.OriginModelName,
		ResultURL:   resultURL,
		FailReason:  failReason,
		CreatedAt:   task.CreatedAt,
		CompletedAt: task.FinishTime,
	}
}

func sanitizePublicTaskData(task *model.Task) json.RawMessage {
	if len(task.Data) == 0 {
		return nil
	}

	var value any
	if err := common.Unmarshal(task.Data, &value); err != nil {
		// 无法解析的上游原始响应不应直接回显给客户。
		return nil
	}

	upstreamIDs := map[string]struct{}{}
	upstreamTaskID := task.GetUpstreamTaskID()
	if upstreamTaskID != "" && upstreamTaskID != task.TaskID {
		upstreamIDs[upstreamTaskID] = struct{}{}
		if decoded, err := taskcommon.DecodeLocalTaskID(upstreamTaskID); err == nil && strings.Contains(decoded, "/operations/") {
			upstreamIDs[decoded] = struct{}{}
		}
	}

	value = sanitizePublicTaskValue(value, task.TaskID, task.Properties.OriginModelName, upstreamIDs, true)
	data, err := common.Marshal(value)
	if err != nil {
		return nil
	}
	return json.RawMessage(data)
}

func sanitizePublicTaskValue(value any, publicTaskID, originModelName string, upstreamIDs map[string]struct{}, root bool) any {
	switch typed := value.(type) {
	case map[string]any:
		cleaned := make(map[string]any, len(typed))
		for key, item := range typed {
			normalizedKey := normalizePublicTaskKey(key)
			if _, sensitive := publicTaskSensitiveKeys[normalizedKey]; sensitive {
				continue
			}

			switch normalizedKey {
			case "taskid":
				cleaned[key] = publicTaskID
			case "model", "modelname", "originmodelname":
				if originModelName != "" {
					cleaned[key] = originModelName
				}
			case "id":
				if root || isPrivateTaskIdentifier(item, upstreamIDs) {
					cleaned[key] = publicTaskID
				} else {
					cleaned[key] = sanitizePublicTaskValue(item, publicTaskID, originModelName, upstreamIDs, false)
				}
			case "name":
				if isPrivateTaskIdentifier(item, upstreamIDs) || isProviderOperationName(item) {
					cleaned[key] = publicTaskID
				} else {
					cleaned[key] = sanitizePublicTaskValue(item, publicTaskID, originModelName, upstreamIDs, false)
				}
			default:
				cleaned[key] = sanitizePublicTaskValue(item, publicTaskID, originModelName, upstreamIDs, false)
			}
		}
		return cleaned
	case []any:
		cleaned := make([]any, len(typed))
		for i, item := range typed {
			cleaned[i] = sanitizePublicTaskValue(item, publicTaskID, originModelName, upstreamIDs, false)
		}
		return cleaned
	case string:
		if _, private := upstreamIDs[typed]; private || strings.Contains(typed, "/operations/") {
			return publicTaskID
		}
		return typed
	default:
		return typed
	}
}

func normalizePublicTaskKey(key string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, key)
}

func isPrivateTaskIdentifier(value any, upstreamIDs map[string]struct{}) bool {
	id, ok := value.(string)
	if !ok {
		return false
	}
	_, private := upstreamIDs[id]
	return private
}

func isProviderOperationName(value any) bool {
	name, ok := value.(string)
	return ok && strings.Contains(name, "/operations/")
}
