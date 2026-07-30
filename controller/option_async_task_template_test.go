package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAsyncTaskUpstreamTemplatesAcceptsParameterMappings(t *testing.T) {
	raw := `[
  {
    "request_path": "/v1/images/generations/async",
    "parameter_mappings": [
      {
        "name": "1:1 2k",
        "when": {
          "size": "1:1",
          "metadata.resolution": "2k"
        },
        "strip_fields": ["metadata.resolution"],
        "set_fields": {
          "parameters.size": "2048x2048"
        }
      }
    ]
  }
]`

	require.NoError(t, validateAsyncTaskUpstreamTemplates(raw))
}

func TestValidateAsyncTaskUpstreamTemplatesRejectsInvalidParameterMappings(t *testing.T) {
	testCases := []struct {
		name    string
		mapping string
		message string
	}{
		{
			name:    "empty when",
			mapping: `{"when": {}, "set_fields": {"size": "2048x2048"}}`,
			message: "when 不能为空",
		},
		{
			name:    "no action",
			mapping: `{"when": {"size": "1:1"}}`,
			message: "必须配置 strip_fields 或 set_fields",
		},
		{
			name:    "invalid condition path",
			mapping: `{"when": {"metadata..resolution": "2k"}, "set_fields": {"size": "2048x2048"}}`,
			message: "字段路径 \"metadata..resolution\" 无效",
		},
		{
			name:    "invalid output path",
			mapping: `{"when": {"size": "1:1"}, "set_fields": {" parameters.size": "2048x2048"}}`,
			message: "字段路径 \" parameters.size\" 无效",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			raw := `[{"request_path":"/v1/images/generations/async","parameter_mappings":[` + testCase.mapping + `]}]`
			err := validateAsyncTaskUpstreamTemplates(raw)
			require.ErrorContains(t, err, testCase.message)
		})
	}
}
