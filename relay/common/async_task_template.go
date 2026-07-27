package common

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/dto"
)

type AsyncTaskTemplateContext struct {
	RequestPath   string
	Model         string
	UpstreamModel string
	TaskID        string
	ChannelID     int
	ChannelType   int
	Body          map[string]any
	APIKey        string
}

func SelectAsyncTaskUpstreamTemplate(
	templates []dto.AsyncTaskUpstreamTemplate,
	channelID int,
	channelType int,
	requestPath string,
	modelName string,
) *dto.AsyncTaskUpstreamTemplate {
	requestPath = normalizeTemplateRequestPath(requestPath)
	modelName = strings.TrimSpace(modelName)
	for i := range templates {
		template := &templates[i]
		if !template.Enabled {
			continue
		}
		if !matchTemplateChannel(template.ChannelID, channelID) {
			continue
		}
		if !matchTemplateChannel(template.ChannelType, channelType) {
			continue
		}
		if !matchTemplatePath(template.RequestPath, requestPath) {
			continue
		}
		if !matchTemplateModel(template.Model, template.NameRule, modelName) {
			continue
		}
		return template
	}
	return nil
}

func BuildAsyncTaskTemplateURL(baseURL string, pathTemplate string, ctx AsyncTaskTemplateContext) string {
	resolvedPath := ResolveAsyncTaskTemplateString(pathTemplate, ctx)
	if parsed, err := url.Parse(resolvedPath); err == nil && parsed.IsAbs() {
		return resolvedPath
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(resolvedPath, "/")
}

func ResolveAsyncTaskTemplateValue(value any, ctx AsyncTaskTemplateContext) any {
	switch v := value.(type) {
	case string:
		if path, ok := exactBodyTemplatePath(v); ok {
			return cloneTemplateValue(getTemplateBodyValue(ctx.Body, path))
		}
		return ResolveAsyncTaskTemplateString(v, ctx)
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = ResolveAsyncTaskTemplateValue(item, ctx)
		}
		return out
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, ResolveAsyncTaskTemplateValue(item, ctx))
		}
		return out
	default:
		return value
	}
}

func ResolveAsyncTaskTemplateString(value string, ctx AsyncTaskTemplateContext) string {
	replacer := strings.NewReplacer(
		"{request_path}", ctx.RequestPath,
		"{model}", ctx.Model,
		"{origin_model}", ctx.Model,
		"{upstream_model}", ctx.UpstreamModel,
		"{task_id}", ctx.TaskID,
		"{api_key}", ctx.APIKey,
	)
	out := replacer.Replace(value)
	for {
		start := strings.Index(out, "{body:")
		if start < 0 {
			break
		}
		end := strings.Index(out[start:], "}")
		if end < 0 {
			break
		}
		end += start
		token := out[start : end+1]
		path := strings.TrimSuffix(strings.TrimPrefix(token, "{body:"), "}")
		out = strings.ReplaceAll(out, token, fmt.Sprint(getTemplateBodyValue(ctx.Body, path)))
	}
	return out
}

func ApplyAsyncTaskTemplateToBody(
	source map[string]any,
	template dto.AsyncTaskUpstreamTemplate,
	ctx AsyncTaskTemplateContext,
) map[string]any {
	var out map[string]any
	if len(template.Body) > 0 {
		if resolved, ok := ResolveAsyncTaskTemplateValue(template.Body, ctx).(map[string]any); ok {
			out = resolved
		} else {
			out = make(map[string]any)
		}
	} else {
		out = cloneTemplateBody(source)
	}

	for _, field := range template.StripFields {
		deleteTemplateBodyField(out, field)
	}
	for field, value := range template.SetFields {
		setTemplateBodyField(out, field, ResolveAsyncTaskTemplateValue(value, ctx))
	}
	if _, ok := out["model"]; !ok && ctx.UpstreamModel != "" {
		out["model"] = ctx.UpstreamModel
	}
	return out
}

func normalizeTemplateRequestPath(value string) string {
	value = strings.TrimSpace(value)
	if idx := strings.Index(value, "?"); idx >= 0 {
		value = value[:idx]
	}
	return strings.TrimRight(value, "/")
}

func matchTemplatePath(rulePath string, requestPath string) bool {
	rulePath = normalizeTemplateRequestPath(rulePath)
	if rulePath == "" {
		return true
	}
	return rulePath == requestPath
}

func matchTemplateChannel(ruleValue int, actualValue int) bool {
	return ruleValue == 0 || ruleValue == actualValue
}

func matchTemplateModel(ruleModel string, nameRule int, modelName string) bool {
	ruleModel = strings.TrimSpace(ruleModel)
	if ruleModel == "" {
		return true
	}
	switch nameRule {
	case 1:
		return strings.HasPrefix(modelName, ruleModel)
	case 2:
		return strings.Contains(modelName, ruleModel)
	case 3:
		return strings.HasSuffix(modelName, ruleModel)
	default:
		return modelName == ruleModel
	}
}

func cloneTemplateBody(source map[string]any) map[string]any {
	out := make(map[string]any, len(source))
	for key, value := range source {
		if nested, ok := value.(map[string]any); ok {
			out[key] = cloneTemplateBody(nested)
		} else {
			out[key] = value
		}
	}
	return out
}

func exactBodyTemplatePath(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "{body:") || !strings.HasSuffix(value, "}") {
		return "", false
	}
	if strings.Count(value, "{body:") != 1 || strings.Count(value, "}") != 1 {
		return "", false
	}
	path := strings.TrimSuffix(strings.TrimPrefix(value, "{body:"), "}")
	if strings.TrimSpace(path) == "" {
		return "", false
	}
	return path, true
}

func getTemplateBodyValue(body map[string]any, path string) any {
	if body == nil {
		return ""
	}
	current := any(body)
	for _, part := range strings.Split(path, ".") {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = m[part]
		if !ok {
			return ""
		}
	}
	return current
}

func cloneTemplateValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return cloneTemplateBody(v)
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, cloneTemplateValue(item))
		}
		return out
	default:
		return value
	}
}

func deleteTemplateBodyField(body map[string]any, path string) {
	parts := strings.Split(strings.TrimSpace(path), ".")
	if len(parts) == 0 || parts[0] == "" {
		return
	}
	current := body
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			return
		}
		current = next
	}
	delete(current, parts[len(parts)-1])
}

func setTemplateBodyField(body map[string]any, path string, value any) {
	parts := strings.Split(strings.TrimSpace(path), ".")
	if len(parts) == 0 || parts[0] == "" {
		return
	}
	current := body
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			next = make(map[string]any)
			current[part] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
}
