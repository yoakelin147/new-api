package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetQuotaDataGroupByChannel(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&Channel{Id: 11, Name: "primary-channel", Key: "key-11"}).Error)
	require.NoError(t, DB.Create(&Channel{Id: 22, Name: "backup-channel", Key: "key-22"}).Error)

	logs := []*Log{
		{
			CreatedAt:        3600 + 10,
			Type:             LogTypeConsume,
			ChannelId:        11,
			Quota:            100,
			PromptTokens:     7,
			CompletionTokens: 3,
		},
		{
			CreatedAt:        3600 + 20,
			Type:             LogTypeConsume,
			ChannelId:        11,
			Quota:            200,
			PromptTokens:     5,
			CompletionTokens: 5,
		},
		{
			CreatedAt:        7200 + 10,
			Type:             LogTypeConsume,
			ChannelId:        22,
			Quota:            400,
			PromptTokens:     20,
			CompletionTokens: 30,
		},
		{
			CreatedAt: 3600 + 30,
			Type:      LogTypeError,
			ChannelId: 11,
			Quota:     999,
		},
		{
			CreatedAt: 3600 + 40,
			Type:      LogTypeConsume,
			Quota:     999,
		},
	}
	require.NoError(t, DB.Create(&logs).Error)

	rows, err := GetQuotaDataGroupByChannel(3600, 7200+3599)
	require.NoError(t, err)

	expected := []*ChannelQuotaData{
		{
			ChannelId:   11,
			ChannelName: "primary-channel",
			CreatedAt:   3600,
			TokenUsed:   20,
			Count:       2,
			Quota:       300,
		},
		{
			ChannelId:   22,
			ChannelName: "backup-channel",
			CreatedAt:   7200,
			TokenUsed:   50,
			Count:       1,
			Quota:       400,
		},
	}
	assert.Equal(t, expected, rows)
}

func TestGetQuotaDataGroupByTokenFiltersCurrentUser(t *testing.T) {
	truncateTables(t)

	logs := []*Log{
		{
			UserId:           7,
			CreatedAt:        3600 + 10,
			Type:             LogTypeConsume,
			TokenId:          101,
			TokenName:        "primary-key",
			Quota:            100,
			PromptTokens:     7,
			CompletionTokens: 3,
		},
		{
			UserId:           7,
			CreatedAt:        3600 + 20,
			Type:             LogTypeConsume,
			TokenId:          101,
			TokenName:        "primary-key",
			Quota:            200,
			PromptTokens:     5,
			CompletionTokens: 5,
		},
		{
			UserId:           7,
			CreatedAt:        7200 + 10,
			Type:             LogTypeConsume,
			TokenId:          202,
			TokenName:        "backup-key",
			Quota:            400,
			PromptTokens:     20,
			CompletionTokens: 30,
		},
		{
			UserId:    8,
			CreatedAt: 3600 + 30,
			Type:      LogTypeConsume,
			TokenId:   303,
			TokenName: "other-user-key",
			Quota:     999,
		},
		{
			UserId:    7,
			CreatedAt: 3600 + 40,
			Type:      LogTypeError,
			TokenId:   101,
			TokenName: "primary-key",
			Quota:     999,
		},
		{
			UserId:    7,
			CreatedAt: 3600 + 50,
			Type:      LogTypeConsume,
			Quota:     999,
		},
	}
	require.NoError(t, DB.Create(&logs).Error)

	rows, err := GetQuotaDataGroupByToken(7, 3600, 7200+3599)
	require.NoError(t, err)

	expected := []*TokenQuotaData{
		{
			TokenId:   101,
			TokenName: "primary-key",
			CreatedAt: 3600,
			TokenUsed: 20,
			Count:     2,
			Quota:     300,
		},
		{
			TokenId:   202,
			TokenName: "backup-key",
			CreatedAt: 7200,
			TokenUsed: 50,
			Count:     1,
			Quota:     400,
		},
	}
	assert.Equal(t, expected, rows)
}
