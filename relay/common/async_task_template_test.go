package common

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestApplyAsyncTaskTemplateToBodyPreservesExactBodyPlaceholderTypes(t *testing.T) {
	source := map[string]any{
		"prompt":  "animate",
		"images":  []any{"https://example.com/a.png", "https://example.com/b.png"},
		"seconds": float64(6),
		"metadata": map[string]any{
			"seed": float64(123),
		},
	}
	template := dto.AsyncTaskUpstreamTemplate{
		Body: map[string]any{
			"prompt":   "{body:prompt}",
			"images":   "{body:images}",
			"seconds":  "{body:seconds}",
			"metadata": "{body:metadata}",
			"label":    "job-{body:prompt}",
		},
	}

	got := ApplyAsyncTaskTemplateToBody(source, template, AsyncTaskTemplateContext{
		Body: source,
	})

	require.Equal(t, "animate", got["prompt"])
	require.Equal(t, []any{"https://example.com/a.png", "https://example.com/b.png"}, got["images"])
	require.Equal(t, float64(6), got["seconds"])
	require.Equal(t, map[string]any{"seed": float64(123)}, got["metadata"])
	require.Equal(t, "job-animate", got["label"])
}

func TestSelectAsyncTaskUpstreamTemplateMatchesAnyCommaSeparatedModel(t *testing.T) {
	template := SelectAsyncTaskUpstreamTemplate(
		[]dto.AsyncTaskUpstreamTemplate{
			{
				Enabled: true,
				Model:   "gpt-image-2, sora-2",
			},
		},
		1,
		1,
		"/v1/videos",
		"sora-2",
	)

	require.NotNil(t, template)

	template = SelectAsyncTaskUpstreamTemplate(
		[]dto.AsyncTaskUpstreamTemplate{
			{
				Enabled: true,
				Model:   "gpt-image-2, sora-2",
			},
		},
		1,
		1,
		"/v1/videos",
		"other-model",
	)

	require.Nil(t, template)
}

func TestApplyAsyncTaskTemplateToBodyPreservesSetFieldPlaceholderTypes(t *testing.T) {
	source := map[string]any{
		"images": []any{"https://example.com/a.png"},
	}
	template := dto.AsyncTaskUpstreamTemplate{
		SetFields: map[string]any{
			"images": "{body:images}",
		},
	}

	got := ApplyAsyncTaskTemplateToBody(source, template, AsyncTaskTemplateContext{
		Body: source,
	})

	require.Equal(t, []any{"https://example.com/a.png"}, got["images"])
}

func TestApplyAsyncTaskTemplateToBodyAppliesFirstMatchingParameterMapping(t *testing.T) {
	source := map[string]any{
		"model":      "gpt-image-2",
		"prompt":     "draw a fox",
		"size":       "1:1",
		"resolution": "2k",
	}
	template := dto.AsyncTaskUpstreamTemplate{
		SetFields: map[string]any{
			"size": "{body:size}",
		},
		ParameterMappings: []dto.AsyncTaskParameterMapping{
			{
				Name: "1:1 2k",
				When: map[string]any{
					"size":       "1:1",
					"resolution": "2k",
				},
				StripFields: []string{"resolution"},
				SetFields: map[string]any{
					"size":             "2048x2048",
					"parameters.ratio": "square",
				},
			},
			{
				Name: "must not run",
				When: map[string]any{"size": "1:1"},
				SetFields: map[string]any{
					"size": "1024x1024",
				},
			},
		},
	}

	result := ApplyAsyncTaskTemplateToBodyWithResult(source, template, AsyncTaskTemplateContext{
		Body: source,
	})

	require.Equal(t, "1:1 2k", result.MatchedParameterMapping.Name)
	require.Equal(t, map[string]any{
		"model":  "gpt-image-2",
		"prompt": "draw a fox",
		"size":   "2048x2048",
		"parameters": map[string]any{
			"ratio": "square",
		},
	}, result.Body)
}

func TestApplyAsyncTaskTemplateToBodyMatchesOriginalNestedValuesStrictly(t *testing.T) {
	source := map[string]any{
		"size": "1:1",
		"metadata": map[string]any{
			"resolution": "2k",
			"count":      float64(2),
		},
	}
	template := dto.AsyncTaskUpstreamTemplate{
		Body: map[string]any{
			"size": "default",
		},
		ParameterMappings: []dto.AsyncTaskParameterMapping{
			{
				When: map[string]any{
					"metadata.resolution": "2k",
					"metadata.count":      float64(2),
				},
				SetFields: map[string]any{"size": "2048x2048"},
			},
		},
	}

	result := ApplyAsyncTaskTemplateToBodyWithResult(source, template, AsyncTaskTemplateContext{
		Body: source,
	})

	require.NotNil(t, result.MatchedParameterMapping)
	require.Equal(t, map[string]any{"size": "2048x2048"}, result.Body)

	source["metadata"].(map[string]any)["count"] = "2"
	result = ApplyAsyncTaskTemplateToBodyWithResult(source, template, AsyncTaskTemplateContext{
		Body: source,
	})
	require.Nil(t, result.MatchedParameterMapping)
	require.Equal(t, map[string]any{"size": "default"}, result.Body)
}

func TestApplyAsyncTaskTemplateToBodyDoesNotMatchMissingFieldAgainstNull(t *testing.T) {
	template := dto.AsyncTaskUpstreamTemplate{
		ParameterMappings: []dto.AsyncTaskParameterMapping{
			{
				When:      map[string]any{"resolution": nil},
				SetFields: map[string]any{"size": "2048x2048"},
			},
		},
	}

	result := ApplyAsyncTaskTemplateToBodyWithResult(map[string]any{}, template, AsyncTaskTemplateContext{
		Body: map[string]any{},
	})

	require.Nil(t, result.MatchedParameterMapping)
}
