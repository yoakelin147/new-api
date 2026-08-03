package billingexpr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractTierUnitPrices(t *testing.T) {
	exprStr := `len <= 272000
? tier("standard", p * 5 + c * 30 + cr * 0.5 + cc * 6.25)
: tier("long_context", p * 10 + c * 45 + cr * 1 + cc * 12.5)`

	standard, ok := ExtractTierUnitPrices(exprStr, "standard")
	assert.True(t, ok)
	assert.Equal(t, UnitPrices{
		Input:      5,
		Output:     30,
		CacheRead:  0.5,
		CacheWrite: 6.25,
	}, standard)

	longContext, ok := ExtractTierUnitPrices(exprStr, "long_context")
	assert.True(t, ok)
	assert.Equal(t, UnitPrices{
		Input:      10,
		Output:     45,
		CacheRead:  1,
		CacheWrite: 12.5,
	}, longContext)
}

func TestExtractTierUnitPricesSupportsLinearArithmetic(t *testing.T) {
	prices, ok := ExtractTierUnitPrices(
		`v1:tier("default", (p * 3 + c * 15 + cr / 10 + cc * 3 + cc1h * 6) * 0.5)`,
		"DEFAULT",
	)

	assert.True(t, ok)
	assert.Equal(t, UnitPrices{
		Input:        1.5,
		Output:       7.5,
		CacheRead:    0.05,
		CacheWrite:   1.5,
		CacheWrite1h: 3,
	}, prices)
}

func TestExtractTierUnitPricesRejectsUnknownOrNonLinearTier(t *testing.T) {
	_, ok := ExtractTierUnitPrices(`tier("default", p * c)`, "default")
	assert.False(t, ok)

	_, ok = ExtractTierUnitPrices(`tier("default", p * 2)`, "missing")
	assert.False(t, ok)
}
