package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateToolPricesJSONAllowsZeroAndRejectsInvalidKeys(t *testing.T) {
	require.NoError(t, ValidateToolPricesJSON(`{"web_search":0,"web_search_preview:gpt-4o*":2.5}`))
	for _, value := range []string{
		`null`,
		`[]`,
		`{"":1}`,
		`{"web search":1}`,
		`{"web_search:gpt-4o":1}`,
		`{"web_search:gpt:*":1}`,
		`{"web_search":-1}`,
		`{"web_search":1e999}`,
	} {
		require.Error(t, ValidateToolPricesJSON(value), value)
	}
}
