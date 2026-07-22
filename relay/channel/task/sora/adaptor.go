package sora

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
)

// ============================
// Request / Response structures
// ============================

type ContentItem struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // for text type
	ImageURL *ImageURL `json:"image_url,omitempty"` // for image_url type
}

type ImageURL struct {
	URL string `json:"url"`
}

type responseTask struct {
	ID                 string `json:"id"`
	TaskID             string `json:"task_id,omitempty"` //兼容旧接口
	Object             string `json:"object"`
	Model              string `json:"model"`
	Status             string `json:"status"`
	Progress           int    `json:"progress"`
	VideoURL           string `json:"video_url,omitempty"`
	URL                string `json:"url,omitempty"`
	ImageURL           string `json:"image_url,omitempty"`
	CreatedAt          int64  `json:"created_at"`
	CompletedAt        int64  `json:"completed_at,omitempty"`
	ExpiresAt          int64  `json:"expires_at,omitempty"`
	Seconds            string `json:"seconds,omitempty"`
	Size               string `json:"size,omitempty"`
	RemixedFromVideoID string `json:"remixed_from_video_id,omitempty"`
	Error              *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType     int
	ChannelID       int
	apiKey          string
	baseURL         string
	responseMapping *dto.AsyncTaskResponseMapping
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
	if info.ChannelMeta != nil {
		a.ChannelID = info.ChannelMeta.ChannelId
	}
}

func validateRemixRequest(c *gin.Context) *dto.TaskError {
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("field prompt is required"), "invalid_request", http.StatusBadRequest)
	}
	// 存储原始请求到 context，与 ValidateMultipartDirect 路径保持一致
	c.Set("task_request", req)
	return nil
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	if info.Action == constant.TaskActionRemix {
		return validateRemixRequest(c)
	}
	return relaycommon.ValidateMultipartDirect(c, info)
}

// EstimateBilling 根据用户请求的 seconds 和 size 计算 OtherRatios。
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	// remix 路径的 OtherRatios 已在 ResolveOriginTask 中设置
	if info.Action == constant.TaskActionRemix {
		return nil
	}
	if info.Action == constant.TaskActionImageGenerate {
		return nil
	}

	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}

	seconds, _ := strconv.Atoi(req.Seconds)
	if seconds == 0 {
		seconds = req.Duration
	}
	if seconds <= 0 {
		seconds = 4
	}

	size := req.Size
	if size == "" {
		size = "720x1280"
	}

	ratios := map[string]float64{
		"seconds": float64(seconds),
		"size":    1,
	}
	if size == "1792x1024" || size == "1024x1792" {
		ratios["size"] = 1.666667
	}
	return ratios
}

func (a *TaskAdaptor) selectUpstreamTemplate(info *relaycommon.RelayInfo) *dto.AsyncTaskUpstreamTemplate {
	if info == nil || info.ChannelMeta == nil {
		return nil
	}
	return relaycommon.SelectAsyncTaskUpstreamTemplate(
		setting.GetAsyncTaskUpstreamTemplates(),
		info.ChannelMeta.ChannelId,
		info.ChannelMeta.ChannelType,
		info.RequestURLPath,
		info.OriginModelName,
	)
}

func (a *TaskAdaptor) templateContext(info *relaycommon.RelayInfo, body map[string]any, taskID string) relaycommon.AsyncTaskTemplateContext {
	if info == nil {
		return relaycommon.AsyncTaskTemplateContext{Body: body}
	}
	if taskID == "" && info.TaskRelayInfo != nil {
		taskID = info.PublicTaskID
	}
	ctx := relaycommon.AsyncTaskTemplateContext{
		RequestPath:   info.RequestURLPath,
		Model:         info.OriginModelName,
		UpstreamModel: info.UpstreamModelName,
		TaskID:        taskID,
		Body:          body,
		APIKey:        a.apiKey,
	}
	if info.ChannelMeta != nil {
		ctx.ChannelID = info.ChannelMeta.ChannelId
		ctx.ChannelType = info.ChannelMeta.ChannelType
	}
	return ctx
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info.Action == constant.TaskActionRemix {
		return fmt.Sprintf("%s/v1/videos/%s/remix", a.baseURL, info.OriginTaskID), nil
	}
	if template := a.selectUpstreamTemplate(info); template != nil && strings.TrimSpace(template.SubmitPath) != "" {
		return relaycommon.BuildAsyncTaskTemplateURL(a.baseURL, template.SubmitPath, a.templateContext(info, nil, "")), nil
	}
	if info.Action == constant.TaskActionImageGenerate {
		return fmt.Sprintf("%s/v1/images/generations/async", a.baseURL), nil
	}
	return fmt.Sprintf("%s/v1/videos", a.baseURL), nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	if template := a.selectUpstreamTemplate(info); template != nil {
		ctx := a.templateContext(info, nil, "")
		for key, value := range template.Headers {
			req.Header.Set(key, relaycommon.ResolveAsyncTaskTemplateString(value, ctx))
		}
	}
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_request_body_failed")
	}
	cachedBody, err := storage.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "read_body_bytes_failed")
	}
	contentType := c.GetHeader("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		var bodyMap map[string]interface{}
		if err := common.Unmarshal(cachedBody, &bodyMap); err == nil {
			if template := a.selectUpstreamTemplate(info); template != nil {
				ctx := a.templateContext(info, bodyMap, "")
				bodyMap = relaycommon.ApplyAsyncTaskTemplateToBody(bodyMap, *template, ctx)
				if _, explicitlySetType := template.SetFields["type"]; !explicitlySetType {
					if _, explicitlyTemplatedType := template.Body["type"]; !explicitlyTemplatedType {
						delete(bodyMap, "type")
					}
				}
			} else {
				bodyMap["model"] = info.UpstreamModelName
				delete(bodyMap, "type")
			}
			if newBody, err := common.Marshal(bodyMap); err == nil {
				return bytes.NewReader(newBody), nil
			}
		}
		return bytes.NewReader(cachedBody), nil
	}

	if strings.Contains(contentType, "multipart/form-data") {
		formData, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return bytes.NewReader(cachedBody), nil
		}
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		writer.WriteField("model", info.UpstreamModelName)
		for key, values := range formData.Value {
			if key == "model" || key == "type" {
				continue
			}
			for _, v := range values {
				writer.WriteField(key, v)
			}
		}
		for fieldName, fileHeaders := range formData.File {
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				ct := fh.Header.Get("Content-Type")
				if ct == "" || ct == "application/octet-stream" {
					buf512 := make([]byte, 512)
					n, _ := io.ReadFull(f, buf512)
					ct = http.DetectContentType(buf512[:n])
					// Re-open after sniffing so the full content is copied below
					f.Close()
					f, err = fh.Open()
					if err != nil {
						continue
					}
				}
				h := make(textproto.MIMEHeader)
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fh.Filename))
				h.Set("Content-Type", ct)
				part, err := writer.CreatePart(h)
				if err != nil {
					f.Close()
					continue
				}
				io.Copy(part, f)
				f.Close()
			}
		}
		writer.Close()
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		return &buf, nil
	}

	return common.ReaderOnly(storage), nil
}

// DoRequest delegates to common helper.
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse handles upstream response, returns taskID etc.
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	// Parse Sora response
	var dResp responseTask
	if err := common.Unmarshal(responseBody, &dResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	var upstreamID string
	var mappedStatus string
	if template := a.selectUpstreamTemplate(info); template != nil {
		upstreamID = relaycommon.ExtractAsyncTaskMappedSubmitID(responseBody, template.ResponseMapping)
		mappedStatus = relaycommon.ExtractAsyncTaskMappedSubmitStatus(responseBody, template.ResponseMapping)
	}
	if upstreamID == "" {
		upstreamID = dResp.ID
	}
	if upstreamID == "" {
		upstreamID = dResp.TaskID
	}
	if upstreamID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	// 使用公开 task_xxxx ID 返回给客户端
	dResp.ID = info.PublicTaskID
	dResp.TaskID = info.PublicTaskID
	if dResp.Object == "" {
		dResp.Object = defaultTaskResponseObject(info)
	}
	if dResp.Model == "" && info != nil {
		dResp.Model = info.OriginModelName
	}
	if dResp.Status == "" {
		dResp.Status = taskStatusToVideoStatus(mappedStatus)
	}
	if dResp.Status == "" {
		dResp.Status = dto.VideoStatusQueued
	}
	if dResp.CreatedAt == 0 {
		dResp.CreatedAt = common.GetTimestamp()
	}
	c.JSON(http.StatusOK, dResp)
	return upstreamID, responseBody, nil
}

func defaultTaskResponseObject(info *relaycommon.RelayInfo) string {
	if info != nil && info.TaskRelayInfo != nil && info.Action == constant.TaskActionImageGenerate {
		return "image"
	}
	return "video"
}

func taskStatusToVideoStatus(status string) string {
	switch model.TaskStatus(status) {
	case model.TaskStatusQueued, model.TaskStatusSubmitted:
		return dto.VideoStatusQueued
	case model.TaskStatusInProgress:
		return dto.VideoStatusInProgress
	case model.TaskStatusSuccess:
		return dto.VideoStatusCompleted
	case model.TaskStatusFailure:
		return dto.VideoStatusFailed
	default:
		return ""
	}
}

// FetchTask fetch task status
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	requestPath, _ := body["request_path"].(string)
	modelName, _ := body["model"].(string)
	channelID, _ := body["channel_id"].(int)
	channelType, _ := body["channel_type"].(int)
	template := relaycommon.SelectAsyncTaskUpstreamTemplate(
		setting.GetAsyncTaskUpstreamTemplates(),
		channelID,
		channelType,
		requestPath,
		modelName,
	)
	a.responseMapping = nil
	if template != nil {
		a.responseMapping = template.ResponseMapping
	}

	ctx := relaycommon.AsyncTaskTemplateContext{
		RequestPath:   requestPath,
		Model:         modelName,
		UpstreamModel: modelName,
		TaskID:        taskID,
		ChannelID:     channelID,
		ChannelType:   channelType,
		Body:          body,
		APIKey:        key,
	}

	uri := fmt.Sprintf("%s/v1/videos/%s", baseUrl, taskID)
	if template != nil && strings.TrimSpace(template.FetchPath) != "" {
		uri = relaycommon.BuildAsyncTaskTemplateURL(baseUrl, template.FetchPath, ctx)
	} else if body["action"] == constant.TaskActionImageGenerate {
		uri = fmt.Sprintf("%s/v1/images/generations/async/%s", baseUrl, taskID)
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+key)
	if template != nil {
		for header, value := range template.Headers {
			req.Header.Set(header, relaycommon.ResolveAsyncTaskTemplateString(value, ctx))
		}
	}

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	if taskResult, ok := relaycommon.ParseAsyncTaskResultWithMapping(respBody, a.responseMapping); ok {
		return taskResult, nil
	}

	resTask := responseTask{}
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{
		Code: 0,
	}

	switch strings.ToLower(strings.TrimSpace(resTask.Status)) {
	case "queued", "pending", "submitted":
		taskResult.Status = model.TaskStatusQueued
	case "processing", "running", "in_progress":
		taskResult.Status = model.TaskStatusInProgress
	case "completed", "succeeded", "success":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Url = extractTaskResultURL(respBody)
	case "failed", "cancelled", "canceled", "error":
		taskResult.Status = model.TaskStatusFailure
		if resTask.Error != nil {
			taskResult.Reason = resTask.Error.Message
		} else {
			taskResult.Reason = "task failed"
		}
	default:
	}
	if resTask.Progress > 0 && resTask.Progress < 100 {
		taskResult.Progress = fmt.Sprintf("%d%%", resTask.Progress)
	} else if taskResult.Status == model.TaskStatusSuccess {
		taskResult.Progress = taskcommon.ProgressComplete
	}

	return &taskResult, nil
}

// extractTaskResultURL 兼容 OpenAI/Sora 以及兔子、Dragon 等常见异步返回。
// 当前第一阶段保存上游直链；后续接 OSS 时，应在任务完成后将该 URL 下载并上传，
// 再把 task.PrivateData.ResultURL 替换为我们自己的 OSS 地址。
func extractTaskResultURL(respBody []byte) string {
	for _, path := range []string{
		"video_url",
		"url",
		"image_url",
		"data.video_url",
		"data.url",
		"data.image_url",
		"data.0.url",
		"data.0.video_url",
		"data.0.image_url",
		"data.result.images.0.url",
		"data.result.images.0.url.0",
		"result.images.0.url",
		"result.images.0.url.0",
		"images.0.url",
		"images.0.url.0",
		"output.0.url",
	} {
		value := gjson.GetBytes(respBody, path)
		if value.Exists() && value.Type == gjson.String {
			if url := strings.TrimSpace(value.String()); url != "" {
				return url
			}
		}
	}
	return ""
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	openAIVideo := task.ToOpenAIVideo()
	openAIVideo.Object = "video"

	var resTask responseTask
	if err := common.Unmarshal(task.Data, &resTask); err == nil && resTask.Error != nil {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: resTask.Error.Message,
			Code:    resTask.Error.Code,
		}
	}

	if openAIVideo.VideoURL == "" {
		openAIVideo.VideoURL = extractTaskResultURL(task.Data)
		if openAIVideo.VideoURL != "" {
			openAIVideo.SetMetadata("url", openAIVideo.VideoURL)
		}
	}

	// 对外永远返回 new-api 公开 task_id。上游 task_id 只保存在 PrivateData，
	// 用于后台轮询和管理员排查，避免客户侧绑定具体供应商。
	openAIVideo.ID = task.TaskID
	openAIVideo.TaskID = task.TaskID
	return common.Marshal(openAIVideo)
}
