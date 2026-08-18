package controller

import (
	"strconv"
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

func TestResolveCostStatisticsPeriodYesterday(t *testing.T) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	start, end, period, err := resolveCostStatisticsPeriod(costStatisticsPeriodYesterday, false, "", "")
	require.NoError(t, err)

	assert.Equal(t, costStatisticsPeriodYesterday, period)
	assert.Equal(t, todayStart.AddDate(0, 0, -1).Unix(), start)
	assert.Equal(t, todayStart.Add(-time.Second).Unix(), end)
}

func TestResolveCostStatisticsPeriodCustomForNonAdmin(t *testing.T) {
	start, end, period, err := resolveCostStatisticsPeriod(costStatisticsPeriodCustom, false, "100", "200")
	require.NoError(t, err)

	assert.Equal(t, costStatisticsPeriodCustom, period)
	assert.Equal(t, int64(100), start)
	assert.Equal(t, int64(200), end)
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

func TestResolveCostStatisticsPeriodCustomAllowsMaximumNonAdminRange(t *testing.T) {
	const startTimestamp int64 = 100
	endTimestamp := startTimestamp + maxNonAdminCostStatisticsCustomRangeSeconds

	start, end, period, err := resolveCostStatisticsPeriod(
		costStatisticsPeriodCustom,
		false,
		strconv.FormatInt(startTimestamp, 10),
		strconv.FormatInt(endTimestamp, 10),
	)
	require.NoError(t, err)

	assert.Equal(t, costStatisticsPeriodCustom, period)
	assert.Equal(t, startTimestamp, start)
	assert.Equal(t, endTimestamp, end)
}

func TestResolveCostStatisticsPeriodCustomRejectsOversizedNonAdminRange(t *testing.T) {
	const startTimestamp int64 = 100
	endTimestamp := startTimestamp + maxNonAdminCostStatisticsCustomRangeSeconds + 1

	_, _, _, err := resolveCostStatisticsPeriod(
		costStatisticsPeriodCustom,
		false,
		strconv.FormatInt(startTimestamp, 10),
		strconv.FormatInt(endTimestamp, 10),
	)
	require.EqualError(t, err, "custom time range cannot exceed 31 days")
}

func TestResolveCostStatisticsPeriodCustomKeepsAdminRangeUnlimited(t *testing.T) {
	const startTimestamp int64 = 100
	endTimestamp := startTimestamp + maxNonAdminCostStatisticsCustomRangeSeconds + 1

	start, end, period, err := resolveCostStatisticsPeriod(
		costStatisticsPeriodCustom,
		true,
		strconv.FormatInt(startTimestamp, 10),
		strconv.FormatInt(endTimestamp, 10),
	)
	require.NoError(t, err)

	assert.Equal(t, costStatisticsPeriodCustom, period)
	assert.Equal(t, startTimestamp, start)
	assert.Equal(t, endTimestamp, end)
}
