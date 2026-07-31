package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const ContextKeySystemAccessKeyId = "system_access_key_id"

func SystemAccessKeyAuth(keyType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")

		rawKey, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			abortSystemAccessKeyUnauthorized(c)
			return
		}

		key, err := model.GetSystemAccessKeyByHash(keyType, model.HashSystemAccessKey(rawKey))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			abortSystemAccessKeyUnauthorized(c)
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"message": "access key validation is unavailable",
			})
			return
		}
		if !key.IsUsable(common.GetTimestamp()) {
			abortSystemAccessKeyUnauthorized(c)
			return
		}

		c.Set(ContextKeySystemAccessKeyId, key.Id)
		if err = model.TouchSystemAccessKey(key.Id, common.GetTimestamp()); err != nil {
			common.SysError("failed to update system access key usage: " + err.Error())
		}
		c.Next()
	}
}

func bearerToken(authorization string) (string, bool) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(authorization), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	if strings.TrimSpace(token) != token || strings.ContainsAny(token, " \t\r\n") {
		return "", false
	}
	return token, true
}

func abortSystemAccessKeyUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"message": "unauthorized",
	})
}
