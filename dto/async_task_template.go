package dto

// ModelEndpointTemplate 是客户侧模型端点模板，用于模型管理页快速生成 endpoints 配置。
type ModelEndpointTemplate struct {
	Path           string              `json:"path"`
	Method         string              `json:"method"`
	Description    string              `json:"description,omitempty"`
	RequestExample map[string]any      `json:"request_example,omitempty"`
	Parameters     []EndpointParameter `json:"parameters,omitempty"`
}

// EndpointParameter 是模型详情 API 页展示的端点参数说明。
type EndpointParameter struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Default     any      `json:"default,omitempty"`
	Range       string   `json:"range,omitempty"`
	EnumValues  []string `json:"enum_values,omitempty"`
	Description string   `json:"description"`
	Required    bool     `json:"required,omitempty"`
}

// AsyncTaskUpstreamTemplate 是内部异步任务上游模板。
// 关键匹配维度为 channel_id/channel_type + request_path + model/name_rule，避免同名模型在不同渠道混用规则。
type AsyncTaskUpstreamTemplate struct {
	Enabled               bool                      `json:"enabled,omitempty"`
	Name                  string                    `json:"name,omitempty"`
	ChannelID             int                       `json:"channel_id,omitempty"`   // 指定渠道 ID；0 表示不限制
	ChannelType           int                       `json:"channel_type,omitempty"` // 指定渠道类型；0 表示不限制
	RequestPath           string                    `json:"request_path,omitempty"` // 客户请求 new-api 的路径，如 /v1/images/generations/async
	Model                 string                    `json:"model,omitempty"`        // 模型匹配值
	NameRule              int                       `json:"name_rule,omitempty"`    // 0 精确，1 前缀，2 包含，3 后缀
	SubmitPath            string                    `json:"submit_path,omitempty"`  // 上游提交路径，支持 {task_id}/{model}/{upstream_model}
	FetchPath             string                    `json:"fetch_path,omitempty"`   // 上游查询路径，支持 {task_id}
	Method                string                    `json:"method,omitempty"`
	Headers               map[string]string         `json:"headers,omitempty"`
	StripFields           []string                  `json:"strip_fields,omitempty"`            // 透传原请求时需要删除的字段
	SetFields             map[string]any            `json:"set_fields,omitempty"`              // 透传原请求时需要覆盖/补充的字段
	Body                  map[string]any            `json:"body,omitempty"`                    // 非空时用该模板生成上游请求体
	ResponseMapping       *AsyncTaskResponseMapping `json:"response_mapping,omitempty"`        // 上游提交/查询回参字段映射
	SubmitResponseExample map[string]any            `json:"submit_response_example,omitempty"` // 后台配置预览用，不参与运行
	FetchResponseExample  map[string]any            `json:"fetch_response_example,omitempty"`  // 后台配置预览用，不参与运行
}

// AsyncTaskResponseMapping 描述异步任务上游响应字段如何映射到本站任务字段。
// 路径使用 gjson 风格，如 data.task_id、data.0.url、result.images.0.url。
type AsyncTaskResponseMapping struct {
	SubmitTaskIDPaths      []string          `json:"submit_task_id_paths,omitempty"`
	SubmitStatusPaths      []string          `json:"submit_status_paths,omitempty"`
	FetchStatusPaths       []string          `json:"fetch_status_paths,omitempty"`
	FetchStatusMap         map[string]string `json:"fetch_status_map,omitempty"`
	FetchProgressPaths     []string          `json:"fetch_progress_paths,omitempty"`
	FetchResultURLPaths    []string          `json:"fetch_result_url_paths,omitempty"`
	FetchErrorMessagePaths []string          `json:"fetch_error_message_paths,omitempty"`
}
