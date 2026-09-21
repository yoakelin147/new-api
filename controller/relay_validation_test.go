package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMoonshotRequestValidationErrorsReturnBadRequest(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(ctx, constant.ContextKeyChannelType, constant.ChannelTypeMoonshot)

	err := requestValidationError(ctx, errors.New("input is required"))

	require.Equal(t, http.StatusBadRequest, err.StatusCode)
	require.Equal(t, "input is required", err.Error())
}

func TestOtherChannelRequestValidationStatusIsUnchanged(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(ctx, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)

	err := requestValidationError(ctx, errors.New("input is required"))

	require.Equal(t, http.StatusInternalServerError, err.StatusCode)
}
