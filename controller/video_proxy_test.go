package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestGetStoredVideoResultURLPrefersDirectResultURL(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public",
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://cdn.example.com/video.mp4",
		},
	}

	require.Equal(t, "https://cdn.example.com/video.mp4", getStoredVideoResultURL(task))
}

func TestGetStoredVideoResultURLIgnoresProxyURL(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public",
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://api.example.com/v1/videos/task_public/content",
		},
	}

	require.Empty(t, getStoredVideoResultURL(task))
}

func TestGetStoredVideoResultURLIgnoresDataURL(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public",
		PrivateData: model.TaskPrivateData{
			ResultURL: "data:video/mp4;base64,AAAA",
		},
	}

	require.Empty(t, getStoredVideoResultURL(task))
}
