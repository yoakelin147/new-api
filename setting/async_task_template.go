package setting

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

const (
	ModelEndpointTemplatesOptionKey     = "ModelEndpointTemplates"
	AsyncTaskUpstreamTemplatesOptionKey = "AsyncTaskUpstreamTemplates"
)

func GetModelEndpointTemplates() map[string]dto.ModelEndpointTemplate {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[ModelEndpointTemplatesOptionKey]
	common.OptionMapRWMutex.RUnlock()

	templates := make(map[string]dto.ModelEndpointTemplate)
	if strings.TrimSpace(raw) == "" {
		return templates
	}
	if err := common.UnmarshalJsonStr(raw, &templates); err != nil {
		common.SysError("parse ModelEndpointTemplates failed: " + err.Error())
		return make(map[string]dto.ModelEndpointTemplate)
	}
	return templates
}

func GetAsyncTaskUpstreamTemplates() []dto.AsyncTaskUpstreamTemplate {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[AsyncTaskUpstreamTemplatesOptionKey]
	common.OptionMapRWMutex.RUnlock()

	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var templates []dto.AsyncTaskUpstreamTemplate
	if err := common.UnmarshalJsonStr(raw, &templates); err != nil {
		common.SysError("parse AsyncTaskUpstreamTemplates failed: " + err.Error())
		return nil
	}
	return templates
}
