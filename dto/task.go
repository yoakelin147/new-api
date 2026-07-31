package dto

import (
	"encoding/json"
)

type TaskError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Data       any    `json:"data"`
	StatusCode int    `json:"-"`
	LocalError bool   `json:"-"`
	Error      error  `json:"-"`
}

type TaskData interface {
	SunoDataResponse | []SunoDataResponse | string | any
}

const TaskSuccessCode = "success"

type TaskResponse[T TaskData] struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func (t *TaskResponse[T]) IsSuccess() bool {
	return t.Code == TaskSuccessCode
}

type TaskDto struct {
	ID         int64           `json:"id"`
	CreatedAt  int64           `json:"created_at"`
	UpdatedAt  int64           `json:"updated_at"`
	TaskID     string          `json:"task_id"`
	Platform   string          `json:"platform"`
	UserId     int             `json:"user_id"`
	Group      string          `json:"group"`
	ChannelId  int             `json:"channel_id"`
	Quota      int             `json:"quota"`
	Action     string          `json:"action"`
	Status     string          `json:"status"`
	FailReason string          `json:"fail_reason"`
	ResultURL  string          `json:"result_url,omitempty"` // 任务结果 URL（视频地址等）
	SubmitTime int64           `json:"submit_time"`
	StartTime  int64           `json:"start_time"`
	FinishTime int64           `json:"finish_time"`
	Progress   string          `json:"progress"`
	Properties any             `json:"properties"`
	Username   string          `json:"username,omitempty"`
	Data       json.RawMessage `json:"data"`
}

// PublicTaskProperties 保存可安全返回给任务提交者的公共属性。
// 上游模型映射等供应商信息仍只在内部使用。
type PublicTaskProperties struct {
	Input           string `json:"input"`
	OriginModelName string `json:"origin_model_name,omitempty"`
}

// PublicTaskDto 是通用任务查询接口使用的客户响应结构。
// 内部 TaskDto 继续供管理后台与内部接口使用。
type PublicTaskDto struct {
	CreatedAt  int64                `json:"created_at"`
	UpdatedAt  int64                `json:"updated_at"`
	TaskID     string               `json:"task_id"`
	Action     string               `json:"action"`
	Status     string               `json:"status"`
	FailReason string               `json:"fail_reason"`
	ResultURL  string               `json:"result_url,omitempty"`
	SubmitTime int64                `json:"submit_time"`
	StartTime  int64                `json:"start_time"`
	FinishTime int64                `json:"finish_time"`
	Progress   string               `json:"progress"`
	Properties PublicTaskProperties `json:"properties"`
	Data       json.RawMessage      `json:"data"`
}

// PublicImageTaskDto 是异步图片任务查询使用的精简客户响应结构。
type PublicImageTaskDto struct {
	TaskID      string `json:"task_id"`
	Status      string `json:"status"`
	Progress    string `json:"progress"`
	Model       string `json:"model,omitempty"`
	ResultURL   string `json:"result_url,omitempty"`
	FailReason  string `json:"fail_reason,omitempty"`
	CreatedAt   int64  `json:"created_at"`
	CompletedAt int64  `json:"completed_at,omitempty"`
}

type FetchReq struct {
	IDs []string `json:"ids"`
}
