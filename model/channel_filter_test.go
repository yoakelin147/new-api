package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestGetRandomSatisfiedChannelWithFilterSkipsHigherPriority(t *testing.T) {
	truncateTables(t)
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
	})

	highPriority := int64(100)
	lowPriority := int64(10)
	weight := uint(100)
	require.NoError(t, DB.Create(&Channel{
		Id:       1,
		Key:      "key-1",
		Status:   common.ChannelStatusEnabled,
		Name:     "sync-only",
		Priority: &highPriority,
		Weight:   &weight,
	}).Error)
	require.NoError(t, DB.Create(&Channel{
		Id:       2,
		Key:      "key-2",
		Status:   common.ChannelStatusEnabled,
		Name:     "async-enabled",
		Priority: &lowPriority,
		Weight:   &weight,
	}).Error)
	for _, ability := range []Ability{
		{
			Group:     "default",
			Model:     "gpt-image-2",
			ChannelId: 1,
			Enabled:   true,
			Priority:  &highPriority,
			Weight:    weight,
		},
		{
			Group:     "default",
			Model:     "gpt-image-2",
			ChannelId: 2,
			Enabled:   true,
			Priority:  &lowPriority,
			Weight:    weight,
		},
	} {
		require.NoError(t, DB.Create(&ability).Error)
	}

	channel, err := GetRandomSatisfiedChannelWithFilter(
		"default",
		"gpt-image-2",
		0,
		func(channel *Channel) bool {
			return channel.Id == 2
		},
	)

	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 2, channel.Id)
}

func TestGetRandomSatisfiedChannelWithFilterReturnsNilWhenNoChannelMatches(t *testing.T) {
	truncateTables(t)
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
	})

	priority := int64(100)
	weight := uint(100)
	require.NoError(t, DB.Create(&Channel{
		Id:       1,
		Key:      "key-1",
		Status:   common.ChannelStatusEnabled,
		Name:     "sync-only",
		Priority: &priority,
		Weight:   &weight,
	}).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "default",
		Model:     "gpt-image-2",
		ChannelId: 1,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
	}).Error)

	channel, err := GetRandomSatisfiedChannelWithFilter(
		"default",
		"gpt-image-2",
		0,
		func(channel *Channel) bool {
			return channel.Id == 2
		},
	)

	require.NoError(t, err)
	require.Nil(t, channel)
}
