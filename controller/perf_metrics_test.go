package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetPerfMetricsChannelSummaryRejectsInvalidParameters(t *testing.T) {
	tests := []string{
		"?minutes=invalid",
		"?minutes=0",
		"?minutes=6",
		"?minutes=1441",
		"?limit=0",
		"?limit=101",
	}

	gin.SetMode(gin.TestMode)
	for _, query := range tests {
		t.Run(query, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/perf-metrics/channel-summary"+query, nil)

			GetPerfMetricsChannelSummary(ctx)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}
