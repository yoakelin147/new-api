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
