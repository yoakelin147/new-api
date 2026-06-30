package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxAIConfigureModels = 50

type modelConfigExportModel struct {
	Description string          `json:"description"`
	Endpoints   json.RawMessage `json:"endpoints,omitempty"`
	Icon        string          `json:"icon"`
	ModelName   string          `json:"model_name"`
	NameRule    int             `json:"name_rule"`
	Status      int             `json:"status"`
	Tags        string          `json:"tags"`
	VendorName  string          `json:"vendor_name"`
}

type modelConfigExportFile struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message"`
	Models  []modelConfigExportModel `json:"models"`
	Vendors []upstreamVendor         `json:"vendors"`
}

type aiConfigureRequest struct {
	ModelNames []string `json:"model_names"`
	TokenID    int      `json:"token_id"`
	Group      string   `json:"group"`
	AIModel    string   `json:"ai_model"`
	Language   string   `json:"language"`
	Apply      bool     `json:"apply"`
}

type aiConfigureApplyResult struct {
	CreatedModels  int      `json:"created_models"`
	CreatedVendors int      `json:"created_vendors"`
	SkippedModels  []string `json:"skipped_models"`
}

type aiConfigureChatRequest struct {
	Model       string        `json:"model"`
	Group       string        `json:"group,omitempty"`
	Messages    []dto.Message `json:"messages"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

func normalizeEndpointRawMessage(raw string) json.RawMessage {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var payload any
	if err := common.UnmarshalJsonStr(raw, &payload); err == nil {
		return json.RawMessage(raw)
	}
	data, err := common.Marshal(raw)
	if err != nil {
		return nil
	}
	return json.RawMessage(data)
}

func buildCurrentModelConfig() (*modelConfigExportFile, error) {
	var modelsMeta []model.Model
	if err := model.DB.Order("model_name ASC").Find(&modelsMeta).Error; err != nil {
		return nil, err
	}

	var vendors []model.Vendor
	if err := model.DB.Order("name ASC").Find(&vendors).Error; err != nil {
		return nil, err
	}
	vendorByID := make(map[int]model.Vendor, len(vendors))
	exportVendors := make([]upstreamVendor, 0, len(vendors))
	for _, v := range vendors {
		vendorByID[v.Id] = v
		exportVendors = append(exportVendors, upstreamVendor{
			Name:        v.Name,
			Description: v.Description,
			Icon:        v.Icon,
			Status:      v.Status,
		})
	}

	exportModels := make([]modelConfigExportModel, 0, len(modelsMeta))
	for _, m := range modelsMeta {
		vendorName := ""
		if v, ok := vendorByID[m.VendorID]; ok {
			vendorName = v.Name
		}
		exportModels = append(exportModels, modelConfigExportModel{
			ModelName:   m.ModelName,
			Description: m.Description,
			Icon:        m.Icon,
			Tags:        m.Tags,
			VendorName:  vendorName,
			NameRule:    m.NameRule,
			Status:      m.Status,
			Endpoints:   normalizeEndpointRawMessage(m.Endpoints),
		})
	}

	return &modelConfigExportFile{
		Success: true,
		Message: "Exported current model metadata configuration",
		Models:  exportModels,
		Vendors: exportVendors,
	}, nil
}

func ExportModelConfig(c *gin.Context) {
	config, err := buildCurrentModelConfig()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="model-metadata-config.json"`)
	c.JSON(http.StatusOK, config)
}

func normalizeRequestedMissingModels(requested []string) ([]string, error) {
	missing, err := model.GetMissingModels()
	if err != nil {
		return nil, err
	}
	missingSet := make(map[string]struct{}, len(missing))
	for _, name := range missing {
		missingSet[name] = struct{}{}
	}

	seen := make(map[string]struct{}, len(requested))
	normalized := make([]string, 0, len(requested))
	for _, name := range requested {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		if _, ok := missingSet[name]; !ok {
			return nil, fmt.Errorf("model %s is not missing", name)
		}
		normalized = append(normalized, name)
	}
	if len(normalized) == 0 {
		return nil, errors.New("model_names is required")
	}
	if len(normalized) > maxAIConfigureModels {
		return nil, fmt.Errorf("at most %d models can be configured at once", maxAIConfigureModels)
	}
	sort.Strings(normalized)
	return normalized, nil
}

func normalizeAIConfigureLanguage(language string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "", "zh", "zh-cn", "zh-tw":
		return "zh", "Chinese"
	case "en":
		return "en", "English"
	case "ja":
		return "ja", "Japanese"
	default:
		return "", ""
	}
}

func buildAIConfigurePrompt(modelNames []string, languageName string) (string, string) {
	systemPrompt := `You generate New API model metadata configuration JSON. Return JSON only, without markdown fences or explanations.
The response schema must be:
{
  "success": true,
  "message": "string",
  "models": [
    {
      "model_name": "string",
      "description": "string",
      "icon": "string",
      "tags": "comma,separated,tags",
      "vendor_name": "string",
      "name_rule": 0,
      "status": 1,
      "endpoints": null
    }
  ],
  "vendors": [
    {
      "name": "string",
      "description": "string",
      "icon": "string",
      "status": 1
    }
  ]
}
Rules:
- Include exactly the requested model_name values and no extra models.
- Write message, model description, vendor description, and tags in ` + languageName + `.
- For Chinese output, prefer concise Chinese tags inferred from model names and capabilities, such as 对话, 推理, 工具, 文件, 多模态, 嵌入, 图像, 语音, 视频, 长上下文, 128K, 200K.
- For English output, use concise English tags such as Chat, Reasoning, Tools, Files, Vision, Embeddings, Image, Audio, Video, Long Context, 128K, 200K.
- For Japanese output, use concise Japanese tags such as 対話, 推論, ツール, ファイル, マルチモーダル, 埋め込み, 画像, 音声, 動画, 長文脈, 128K, 200K.
- Keep model_name, vendor_name, and icon as technical identifiers; do not translate those identifiers.
- Use name_rule 0 unless the requested name is clearly a prefix/suffix/contains rule.
- Infer vendor_name, icon, endpoints and tags from public model naming conventions.
- Use conservative descriptions when exact details are uncertain.
- endpoints is optional model marketplace metadata, not the runtime API path list.
- Leave endpoints as null for normal chat/completions models, because New API can infer endpoint support from enabled channels.
- Only set endpoints when the model is clearly non-chat or needs an explicit marketplace endpoint override.
- When endpoints is needed, it must be a JSON object keyed by endpoint type, never an array of API paths.
- Allowed endpoint type keys are openai, openai-response, openai-response-compact, anthropic, gemini, jina-rerank, image-generation, embeddings, openai-video.
- Endpoint object example: {"embeddings":{"path":"/v1/embeddings","method":"POST"}} or {"image-generation":{"path":"/v1/images/generations","method":"POST"}}.
- Deduplicate vendors.`
	userPrompt := "Generate metadata configuration for these missing New API models:\n" + strings.Join(modelNames, "\n")
	return systemPrompt, userPrompt
}

func extractJSONText(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
	}
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		return strings.TrimSpace(text[start : end+1])
	}
	return text
}

func defaultEndpointInfoByType(endpointType constant.EndpointType) (common.EndpointInfo, bool) {
	if endpointType == constant.EndpointTypeOpenAIVideo {
		return common.EndpointInfo{Path: "/v1/videos/generations", Method: "POST"}, true
	}
	return common.GetDefaultEndpointInfo(endpointType)
}

func endpointTypeFromPathOrName(value string) (constant.EndpointType, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	allowed := []constant.EndpointType{
		constant.EndpointTypeOpenAI,
		constant.EndpointTypeOpenAIResponse,
		constant.EndpointTypeOpenAIResponseCompact,
		constant.EndpointTypeAnthropic,
		constant.EndpointTypeGemini,
		constant.EndpointTypeJinaRerank,
		constant.EndpointTypeImageGeneration,
		constant.EndpointTypeEmbeddings,
		constant.EndpointTypeOpenAIVideo,
	}
	for _, endpointType := range allowed {
		if value == string(endpointType) {
			return endpointType, true
		}
		info, ok := defaultEndpointInfoByType(endpointType)
		if ok && value == info.Path {
			return endpointType, true
		}
	}
	switch value {
	case "/v1/rerank", "/rerank":
		return constant.EndpointTypeJinaRerank, true
	case "/v1/images/generations":
		return constant.EndpointTypeImageGeneration, true
	case "/v1/audio/speech", "/v1/audio/transcriptions":
		return constant.EndpointTypeOpenAI, true
	default:
		return "", false
	}
}

func endpointObjectForTypes(endpointTypes []constant.EndpointType) json.RawMessage {
	if len(endpointTypes) == 0 {
		return nil
	}
	config := make(map[string]common.EndpointInfo, len(endpointTypes))
	for _, endpointType := range endpointTypes {
		info, ok := defaultEndpointInfoByType(endpointType)
		if !ok {
			continue
		}
		config[string(endpointType)] = info
	}
	if len(config) == 0 {
		return nil
	}
	data, err := common.Marshal(config)
	if err != nil {
		return nil
	}
	return data
}

func normalizeAIEndpointMetadata(endpoints json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(endpoints)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}

	var object map[string]any
	if err := common.Unmarshal(trimmed, &object); err == nil {
		normalizedTypes := make([]constant.EndpointType, 0, len(object))
		seen := make(map[constant.EndpointType]struct{}, len(object))
		for key := range object {
			if endpointType, ok := endpointTypeFromPathOrName(key); ok {
				if _, exists := seen[endpointType]; exists {
					continue
				}
				seen[endpointType] = struct{}{}
				normalizedTypes = append(normalizedTypes, endpointType)
			}
		}
		return endpointObjectForTypes(normalizedTypes)
	}

	var list []string
	if err := common.Unmarshal(trimmed, &list); err == nil {
		normalizedTypes := make([]constant.EndpointType, 0, len(list))
		seen := make(map[constant.EndpointType]struct{}, len(list))
		for _, item := range list {
			endpointType, ok := endpointTypeFromPathOrName(item)
			if !ok {
				continue
			}
			if _, exists := seen[endpointType]; exists {
				continue
			}
			seen[endpointType] = struct{}{}
			normalizedTypes = append(normalizedTypes, endpointType)
		}
		return endpointObjectForTypes(normalizedTypes)
	}

	var value string
	if err := common.Unmarshal(trimmed, &value); err == nil {
		if endpointType, ok := endpointTypeFromPathOrName(value); ok {
			return endpointObjectForTypes([]constant.EndpointType{endpointType})
		}
	}
	return nil
}

func normalizeGeneratedModelConfig(config *upstreamConfigFile) {
	for i := range config.Models {
		config.Models[i].Endpoints = normalizeAIEndpointMetadata(config.Models[i].Endpoints)
	}
}

func callAIModelConfig(ctx context.Context, token *model.Token, req aiConfigureRequest, modelNames []string) (*upstreamConfigFile, error) {
	_, languageName := normalizeAIConfigureLanguage(req.Language)
	systemPrompt, userPrompt := buildAIConfigurePrompt(modelNames, languageName)
	chatReq := aiConfigureChatRequest{
		Model: strings.TrimSpace(req.AIModel),
		Group: strings.TrimSpace(req.Group),
		Messages: []dto.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.2,
		Stream:      false,
	}
	if chatReq.Model == "" {
		return nil, errors.New("ai_model is required")
	}

	body, err := common.Marshal(chatReq)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimRight(system_setting.ServerAddress, "/")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token.GetFullKey())
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time.Duration(common.GetEnvOrDefault("MODEL_AI_CONFIG_TIMEOUT_SECONDS", 90)) * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, int64(common.GetEnvOrDefault("MODEL_AI_CONFIG_MAX_MB", 2))<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("AI request failed: %s", common.LocalLogPreview(string(respBody)))
	}

	var textResp dto.OpenAITextResponse
	if err := common.Unmarshal(respBody, &textResp); err != nil {
		return nil, err
	}
	if len(textResp.Choices) == 0 {
		return nil, errors.New("AI response contains no choices")
	}
	content := textResp.Choices[0].Message.StringContent()
	if strings.TrimSpace(content) == "" {
		return nil, errors.New("AI response content is empty")
	}

	var config upstreamConfigFile
	if err := common.Unmarshal([]byte(extractJSONText(content)), &config); err != nil {
		return nil, fmt.Errorf("failed to parse AI JSON: %w", err)
	}
	normalizeGeneratedModelConfig(&config)
	return &config, nil
}

func validateGeneratedConfig(config *upstreamConfigFile, requested []string) error {
	var modelsEnv upstreamEnvelope[upstreamModel]
	var vendorsEnv upstreamEnvelope[upstreamVendor]
	payload, err := common.Marshal(config)
	if err != nil {
		return err
	}
	if err := decodeConfigPayload(string(payload), &modelsEnv, &vendorsEnv); err != nil {
		return err
	}
	if err := validateUpstreamModels(modelsEnv.Data); err != nil {
		return err
	}
	if err := validateUpstreamVendors(vendorsEnv.Data); err != nil {
		return err
	}
	requestedSet := make(map[string]struct{}, len(requested))
	for _, name := range requested {
		requestedSet[name] = struct{}{}
	}
	generatedSet := make(map[string]struct{}, len(modelsEnv.Data))
	for _, m := range modelsEnv.Data {
		generatedSet[m.ModelName] = struct{}{}
		if _, ok := requestedSet[m.ModelName]; !ok {
			return fmt.Errorf("AI returned unexpected model %s", m.ModelName)
		}
	}
	for _, name := range requested {
		if _, ok := generatedSet[name]; !ok {
			return fmt.Errorf("AI did not return model %s", name)
		}
	}
	return nil
}

func applyGeneratedModelConfig(config *upstreamConfigFile) (aiConfigureApplyResult, error) {
	result := aiConfigureApplyResult{
		SkippedModels: []string{},
	}
	vendorIDCache := make(map[string]int)
	vendorByName := buildVendorMap(config.Vendors)

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		for _, up := range config.Models {
			var existing model.Model
			if err := tx.Where("model_name = ?", up.ModelName).First(&existing).Error; err == nil {
				result.SkippedModels = append(result.SkippedModels, up.ModelName)
				continue
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			vendorID, err := ensureVendorIDInTx(tx, up.VendorName, vendorByName, vendorIDCache, &result.CreatedVendors)
			if err != nil {
				return err
			}
			mi := &model.Model{
				ModelName:    up.ModelName,
				Description:  up.Description,
				Icon:         up.Icon,
				Tags:         up.Tags,
				VendorID:     vendorID,
				Endpoints:    common.JsonRawMessageToString(up.Endpoints),
				Status:       chooseStatus(up.Status, 1),
				SyncOfficial: 1,
				NameRule:     up.NameRule,
			}
			if err := mi.Insert(); err != nil {
				return err
			}
			result.CreatedModels++
		}
		return nil
	})
	if err == nil && result.CreatedModels > 0 {
		model.RefreshPricing()
	}
	return result, err
}

func ensureVendorIDInTx(tx *gorm.DB, vendorName string, vendorByName map[string]upstreamVendor, vendorIDCache map[string]int, createdVendors *int) (int, error) {
	vendorName = strings.TrimSpace(vendorName)
	if vendorName == "" {
		return 0, nil
	}
	if id, ok := vendorIDCache[vendorName]; ok {
		return id, nil
	}
	var existing model.Vendor
	if err := tx.Where("name = ?", vendorName).First(&existing).Error; err == nil {
		vendorIDCache[vendorName] = existing.Id
		return existing.Id, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	uv := vendorByName[vendorName]
	v := &model.Vendor{
		Name:        vendorName,
		Description: uv.Description,
		Icon:        coalesce(uv.Icon, ""),
		Status:      chooseStatus(uv.Status, 1),
	}
	now := common.GetTimestamp()
	v.CreatedTime = now
	v.UpdatedTime = now
	if err := tx.Create(v).Error; err != nil {
		return 0, err
	}
	*createdVendors++
	vendorIDCache[vendorName] = v.Id
	return v.Id, nil
}

func AIConfigureMissingModels(c *gin.Context) {
	var req aiConfigureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	modelNames, err := normalizeRequestedMissingModels(req.ModelNames)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if language, _ := normalizeAIConfigureLanguage(req.Language); language == "" {
		common.ApiErrorMsg(c, "language must be zh, en, or ja")
		return
	} else {
		req.Language = language
	}
	token, err := model.GetTokenByIds(req.TokenID, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if token.Status != common.TokenStatusEnabled {
		common.ApiErrorMsg(c, "selected token is not enabled")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(common.GetEnvOrDefault("MODEL_AI_CONFIG_TIMEOUT_SECONDS", 90))*time.Second)
	defer cancel()

	config, err := callAIModelConfig(ctx, token, req, modelNames)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := validateGeneratedConfig(config, modelNames); err != nil {
		common.ApiError(c, err)
		return
	}

	data := gin.H{"config": config}
	if req.Apply {
		applyResult, err := applyGeneratedModelConfig(config)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		data["apply_result"] = applyResult
	}
	common.ApiSuccess(c, data)
}
