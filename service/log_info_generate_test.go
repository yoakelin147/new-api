package service

import (
	"encoding/base64"
	"testing"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
)

func TestInjectTieredBillingInfoIncludesUnitPrices(t *testing.T) {
	exprStr := `tier("standard", p * 5 + c * 30 + cr * 0.5 + cc * 6.25 + cc1h * 10)`
	other := make(map[string]interface{})
	relayInfo := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{ExprString: exprStr},
	}

	InjectTieredBillingInfo(other, relayInfo, &billingexpr.TieredResult{MatchedTier: "standard"})

	assert.Equal(t, map[string]interface{}{
		"billing_mode":              "tiered_expr",
		"expr_b64":                  base64.StdEncoding.EncodeToString([]byte(exprStr)),
		"matched_tier":              "standard",
		"unit_prices_resolved":      true,
		"input_unit_price":          5.0,
		"output_unit_price":         30.0,
		"cache_read_unit_price":     0.5,
		"cache_write_unit_price":    6.25,
		"cache_write_1h_unit_price": 10.0,
	}, other)
}
