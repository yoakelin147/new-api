package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractTokenCacheUsageUsesProviderSpecificDenominator(t *testing.T) {
	tests := []struct {
		name         string
		other        string
		promptTokens int64
		expected     tokenCacheUsage
	}{
		{
			name:         "openai cached tokens are part of prompt tokens",
			other:        `{"cache_tokens":40}`,
			promptTokens: 100,
			expected: tokenCacheUsage{
				CacheTokens:      40,
				CacheWriteTokens: 0,
				CacheInputTokens: 100,
			},
		},
		{
			name:         "claude cache read tokens are outside input tokens",
			other:        `{"usage_semantic":"anthropic","cache_tokens":80,"cache_write_tokens":10}`,
			promptTokens: 10,
			expected: tokenCacheUsage{
				CacheTokens:      80,
				CacheWriteTokens: 10,
				CacheInputTokens: 100,
			},
		},
		{
			name:         "legacy claude logs use split cache creation tokens",
			other:        `{"claude":true,"cache_tokens":80,"cache_creation_tokens_5m":7,"cache_creation_tokens_1h":3}`,
			promptTokens: 10,
			expected: tokenCacheUsage{
				CacheTokens:      80,
				CacheWriteTokens: 10,
				CacheInputTokens: 100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractTokenCacheUsage(tt.other, tt.promptTokens))
		})
	}
}
