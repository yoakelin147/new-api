package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCostStatisticsPeriodLastMonth(t *testing.T) {
	now := time.Now()
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	start, end, period, err := resolveCostStatisticsPeriod(costStatisticsPeriodLastMonth, false, "", "")
	require.NoError(t, err)

	assert.Equal(t, costStatisticsPeriodLastMonth, period)
	assert.Equal(t, thisMonthStart.AddDate(0, -1, 0).Unix(), start)
	assert.Equal(t, thisMonthStart.Add(-time.Second).Unix(), end)
}

func TestResolveCostStatisticsPeriodCustomRequiresAdmin(t *testing.T) {
	start, end, period, err := resolveCostStatisticsPeriod(costStatisticsPeriodCustom, false, "100", "200")
	require.NoError(t, err)

	assert.Equal(t, costStatisticsPeriodToday, period)
	assert.NotEqual(t, int64(100), start)
	assert.NotEqual(t, int64(200), end)
}

func TestResolveCostStatisticsPeriodCustomForAdmin(t *testing.T) {
	start, end, period, err := resolveCostStatisticsPeriod(costStatisticsPeriodCustom, true, "100", "200")
	require.NoError(t, err)

	assert.Equal(t, costStatisticsPeriodCustom, period)
	assert.Equal(t, int64(100), start)
	assert.Equal(t, int64(200), end)
}

func TestResolveCostStatisticsPeriodCustomRejectsInvalidRange(t *testing.T) {
	_, _, _, err := resolveCostStatisticsPeriod(costStatisticsPeriodCustom, true, "200", "100")
	require.Error(t, err)
}
