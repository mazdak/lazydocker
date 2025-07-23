package commands

import (
	"testing"

	"github.com/jesseduffield/lazydocker/pkg/config"
	"github.com/jesseduffield/lazydocker/pkg/i18n"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestAppleContainerCommandCreation(t *testing.T) {
	log := logrus.NewEntry(logrus.New())

	// Create a basic config
	appConfig := &config.AppConfig{
		Runtime: "apple",
	}

	// Create a basic OS command (this won't actually execute commands in tests)
	osCommand := &OSCommand{}

	// Create translation set
	tr := &i18n.TranslationSet{}

	errorChan := make(chan error, 1)

	// Test that NewAppleContainerCommand returns an error when Apple Container is not available
	// (which should be the case in the test environment)
	_, err := NewAppleContainerCommand(log, osCommand, tr, appConfig, errorChan)

	// We expect an error because Apple Container CLI is not available in test environment
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Apple Container CLI not found")
}

func TestIsAppleContainerAvailable(t *testing.T) {
	// Test that isAppleContainerAvailable correctly detects absence of Apple Container
	// In test environment, this should return false
	available := isAppleContainerAvailable()

	// Should be false in test environment since Apple Container is not installed
	assert.False(t, available, "Apple Container should not be available in test environment")
}

func TestParseContainerList(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	appConfig := &config.AppConfig{Runtime: "apple"}
	osCommand := &OSCommand{}
	tr := &i18n.TranslationSet{}
	errorChan := make(chan error, 1)

	// Create command instance for testing parsing methods
	cmd := &AppleContainerCommand{
		Log:       log,
		OSCommand: osCommand,
		Tr:        tr,
		Config:    appConfig,
		ErrorChan: errorChan,
	}

	tests := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{
			name:     "empty output",
			input:    "",
			expected: 0,
			hasError: false,
		},
		{
			name:     "single container",
			input:    `{"id":"abc123","name":"test-container","image":"nginx","state":"running"}`,
			expected: 1,
			hasError: false,
		},
		{
			name: "multiple containers",
			input: `{"id":"abc123","name":"test-container-1","image":"nginx","state":"running"}
{"id":"def456","name":"test-container-2","image":"redis","state":"stopped"}`,
			expected: 2,
			hasError: false,
		},
		{
			name:     "invalid json",
			input:    `{"invalid json}`,
			expected: 0,
			hasError: false, // Should skip invalid lines, not error
		},
		{
			name:     "missing required fields",
			input:    `{"image":"nginx","state":"running"}`,
			expected: 0,
			hasError: false, // Should skip containers with missing required fields
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			containers, err := cmd.parseContainerList(tt.input)

			if tt.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expected, len(containers))
			}
		})
	}
}

func TestParseImageList(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	appConfig := &config.AppConfig{Runtime: "apple"}
	osCommand := &OSCommand{}
	tr := &i18n.TranslationSet{}
	errorChan := make(chan error, 1)

	cmd := &AppleContainerCommand{
		Log:       log,
		OSCommand: osCommand,
		Tr:        tr,
		Config:    appConfig,
		ErrorChan: errorChan,
	}

	tests := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{
			name:     "empty output",
			input:    "",
			expected: 0,
			hasError: false,
		},
		{
			name:     "single image",
			input:    `{"id":"img123","name":"nginx","tag":"latest"}`,
			expected: 1,
			hasError: false,
		},
		{
			name: "multiple images",
			input: `{"id":"img123","name":"nginx","tag":"latest"}
{"id":"img456","name":"redis","tag":"6-alpine"}`,
			expected: 2,
			hasError: false,
		},
		{
			name:     "missing required fields",
			input:    `{"name":"nginx","tag":"latest"}`,
			expected: 0,
			hasError: false, // Should skip images with missing ID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images, err := cmd.parseImageList(tt.input)

			if tt.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expected, len(images))
			}
		})
	}
}

func TestJsonToContainer(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	appConfig := &config.AppConfig{Runtime: "apple"}
	osCommand := &OSCommand{}
	tr := &i18n.TranslationSet{}
	errorChan := make(chan error, 1)

	cmd := &AppleContainerCommand{
		Log:       log,
		OSCommand: osCommand,
		Tr:        tr,
		Config:    appConfig,
		ErrorChan: errorChan,
	}

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected *Container
	}{
		{
			name: "valid container data",
			input: map[string]interface{}{
				"id":    "abc123",
				"name":  "test-container",
				"image": "nginx:latest",
				"state": "running",
			},
			expected: &Container{
				ID:   "abc123",
				Name: "test-container",
			},
		},
		{
			name: "missing id",
			input: map[string]interface{}{
				"name":  "test-container",
				"image": "nginx:latest",
				"state": "running",
			},
			expected: nil,
		},
		{
			name: "missing name",
			input: map[string]interface{}{
				"id":    "abc123",
				"image": "nginx:latest",
				"state": "running",
			},
			expected: nil,
		},
		{
			name: "state mapping",
			input: map[string]interface{}{
				"id":    "abc123",
				"name":  "test-container",
				"image": "nginx:latest",
				"state": "stopped", // Should be mapped to "exited"
			},
			expected: &Container{
				ID:   "abc123",
				Name: "test-container",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := cmd.jsonToContainer(tt.input)

			if tt.expected == nil {
				assert.Nil(t, container)
			} else {
				assert.NotNil(t, container)
				assert.Equal(t, tt.expected.ID, container.ID)
				assert.Equal(t, tt.expected.Name, container.Name)
			}
		})
	}
}

func TestJsonToImage(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	appConfig := &config.AppConfig{Runtime: "apple"}
	osCommand := &OSCommand{}
	tr := &i18n.TranslationSet{}
	errorChan := make(chan error, 1)

	cmd := &AppleContainerCommand{
		Log:       log,
		OSCommand: osCommand,
		Tr:        tr,
		Config:    appConfig,
		ErrorChan: errorChan,
	}

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected *Image
	}{
		{
			name: "valid image data",
			input: map[string]interface{}{
				"id":   "img123",
				"name": "nginx",
				"tag":  "latest",
			},
			expected: &Image{
				ID:   "img123",
				Name: "nginx",
				Tag:  "latest",
			},
		},
		{
			name: "missing id",
			input: map[string]interface{}{
				"name": "nginx",
				"tag":  "latest",
			},
			expected: nil,
		},
		{
			name: "partial data",
			input: map[string]interface{}{
				"id":   "img123",
				"name": "nginx",
				// missing tag is ok
			},
			expected: &Image{
				ID:   "img123",
				Name: "nginx",
				Tag:  "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image := cmd.jsonToImage(tt.input)

			if tt.expected == nil {
				assert.Nil(t, image)
			} else {
				assert.NotNil(t, image)
				assert.Equal(t, tt.expected.ID, image.ID)
				assert.Equal(t, tt.expected.Name, image.Name)
				assert.Equal(t, tt.expected.Tag, image.Tag)
			}
		})
	}
}
