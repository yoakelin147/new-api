package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchCodexModelsUsesOAuthHeadersAndDeduplicates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "Bearer access-token", request.Header.Get("Authorization"))
		require.Equal(t, "account-1", request.Header.Get("ChatGPT-Account-Id"))
		require.Equal(t, "codex-cli/1.2.3", request.Header.Get("User-Agent"))
		require.Equal(t, "1.2.3", request.URL.Query().Get("client_version"))
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"models":[{"slug":"codex-a"},{"slug":"codex-a"},{"slug":""}]}`))
	}))
	defer server.Close()

	status, models, err := FetchCodexModels(context.Background(), server.Client(), server.URL, &CodexOAuthKey{
		AccessToken: "access-token",
		AccountID:   "account-1",
	}, "1.2.3")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, []string{"codex-a"}, models)
}
