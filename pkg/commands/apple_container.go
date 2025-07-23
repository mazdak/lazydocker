package commands

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jesseduffield/lazydocker/pkg/config"
	"github.com/jesseduffield/lazydocker/pkg/i18n"
	"github.com/sirupsen/logrus"
)

// AppleContainerCommand handles interactions with Apple's container CLI
type AppleContainerCommand struct {
	Log       *logrus.Entry
	OSCommand *OSCommand
	Tr        *i18n.TranslationSet
	Config    *config.AppConfig
	ErrorChan chan error
}

// NewAppleContainerCommand creates a new Apple Container command handler
func NewAppleContainerCommand(log *logrus.Entry, osCommand *OSCommand, tr *i18n.TranslationSet, config *config.AppConfig, errorChan chan error) (*AppleContainerCommand, error) {
	// Check if Apple Container CLI is available
	if !isAppleContainerAvailable() {
		return nil, fmt.Errorf("Apple Container CLI not found. Please ensure 'container' command is available in PATH")
	}

	return &AppleContainerCommand{
		Log:       log,
		OSCommand: osCommand,
		Tr:        tr,
		Config:    config,
		ErrorChan: errorChan,
	}, nil
}

// isAppleContainerAvailable checks if the Apple Container CLI is available
func isAppleContainerAvailable() bool {
	_, err := exec.LookPath("container")
	return err == nil
}

// GetContainers retrieves all containers from Apple Container
func (c *AppleContainerCommand) GetContainers() ([]*Container, error) {
	c.Log.Info("Getting containers from Apple Container")

	// Execute: container ps --format json
	output, err := c.OSCommand.RunCommandWithOutput("container ps --format json")
	if err != nil {
		c.Log.Error("Failed to get containers from Apple Container: ", err)
		return nil, fmt.Errorf("failed to get containers: %w", err)
	}

	// Parse the JSON output
	containers, err := c.parseContainerList(output)
	if err != nil {
		c.Log.Error("Failed to parse container list: ", err)
		return nil, fmt.Errorf("failed to parse container list: %w", err)
	}

	c.Log.Infof("Found %d containers", len(containers))
	return containers, nil
}

// parseContainerList parses the JSON output from Apple Container's ps command
func (c *AppleContainerCommand) parseContainerList(output string) ([]*Container, error) {
	if strings.TrimSpace(output) == "" {
		return []*Container{}, nil
	}

	// Apple Container might output multiple JSON objects, one per line
	lines := strings.Split(strings.TrimSpace(output), "\n")
	containers := make([]*Container, 0, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var containerData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &containerData); err != nil {
			c.Log.Warn("Failed to parse container JSON line: ", line, " error: ", err)
			continue
		}

		container := c.jsonToContainer(containerData)
		if container != nil {
			containers = append(containers, container)
		}
	}

	return containers, nil
}

// jsonToContainer converts JSON data to a Container struct
func (c *AppleContainerCommand) jsonToContainer(data map[string]interface{}) *Container {
	// Extract basic container information
	id, _ := data["id"].(string)
	name, _ := data["name"].(string)
	image, _ := data["image"].(string)
	state, _ := data["state"].(string)

	if id == "" || name == "" {
		c.Log.Warn("Container missing required fields (id or name)")
		return nil
	}

	// Create container with Apple Container specific fields
	container := &Container{
		ID:        id,
		Name:      name,
		OSCommand: c.OSCommand,
		Log:       c.Log,
		Tr:        c.Tr,
		// Note: We'll implement AppleContainerCommand interface later
	}

	// TODO: Set up container.Details and container.Container properly for Apple Container
	// For now, we'll leave these empty and implement them later when we need specific fields

	// Map Apple Container states to Docker-like states for consistency
	mappedState := state
	switch state {
	case "stopped":
		mappedState = "exited" // Map to Docker terminology for consistency
	}

	c.Log.Debugf("Parsed container: ID=%s, Name=%s, Image=%s, State=%s->%s", id, name, image, state, mappedState)
	return container
}

// BuildImage builds a container image using Apple Container
func (c *AppleContainerCommand) BuildImage(tag, dockerfile string) error {
	c.Log.Infof("Building image with tag %s using dockerfile %s", tag, dockerfile)

	cmd := fmt.Sprintf("container build --tag %s --file %s .", tag, dockerfile)
	return c.OSCommand.RunCommand(cmd)
}

// RunContainer runs a new container using Apple Container
func (c *AppleContainerCommand) RunContainer(name, image string, detached bool) error {
	c.Log.Infof("Running container %s from image %s", name, image)

	cmd := fmt.Sprintf("container run --name %s", name)
	if detached {
		cmd += " --detach"
	}
	cmd += " " + image

	return c.OSCommand.RunCommand(cmd)
}

// StopContainer stops a running container
func (c *AppleContainerCommand) StopContainer(nameOrID string) error {
	c.Log.Infof("Stopping container %s", nameOrID)

	cmd := fmt.Sprintf("container stop %s", nameOrID)
	return c.OSCommand.RunCommand(cmd)
}

// RemoveContainer removes a container
func (c *AppleContainerCommand) RemoveContainer(nameOrID string, force bool) error {
	c.Log.Infof("Removing container %s (force: %v)", nameOrID, force)

	cmd := fmt.Sprintf("container rm")
	if force {
		cmd += " --force"
	}
	cmd += " " + nameOrID

	return c.OSCommand.RunCommand(cmd)
}

// ExecCommand executes a command in a running container
func (c *AppleContainerCommand) ExecCommand(nameOrID, command string) error {
	c.Log.Infof("Executing command in container %s: %s", nameOrID, command)

	cmd := fmt.Sprintf("container exec %s %s", nameOrID, command)
	return c.OSCommand.RunCommand(cmd)
}

// GetImages retrieves all images from Apple Container
func (c *AppleContainerCommand) GetImages() ([]*Image, error) {
	c.Log.Info("Getting images from Apple Container")

	// Execute: container images list --format json
	output, err := c.OSCommand.RunCommandWithOutput("container images list --format json")
	if err != nil {
		c.Log.Error("Failed to get images from Apple Container: ", err)
		return nil, fmt.Errorf("failed to get images: %w", err)
	}

	// Parse the JSON output (implementation similar to containers)
	images, err := c.parseImageList(output)
	if err != nil {
		c.Log.Error("Failed to parse image list: ", err)
		return nil, fmt.Errorf("failed to parse image list: %w", err)
	}

	c.Log.Infof("Found %d images", len(images))
	return images, nil
}

// parseImageList parses the JSON output from Apple Container's images list command
func (c *AppleContainerCommand) parseImageList(output string) ([]*Image, error) {
	if strings.TrimSpace(output) == "" {
		return []*Image{}, nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	images := make([]*Image, 0, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var imageData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &imageData); err != nil {
			c.Log.Warn("Failed to parse image JSON line: ", line, " error: ", err)
			continue
		}

		image := c.jsonToImage(imageData)
		if image != nil {
			images = append(images, image)
		}
	}

	return images, nil
}

// jsonToImage converts JSON data to an Image struct
func (c *AppleContainerCommand) jsonToImage(data map[string]interface{}) *Image {
	id, _ := data["id"].(string)
	name, _ := data["name"].(string)
	tag, _ := data["tag"].(string)

	if id == "" {
		c.Log.Warn("Image missing required ID field")
		return nil
	}

	image := &Image{
		ID:        id,
		Name:      name,
		Tag:       tag,
		OSCommand: c.OSCommand,
		Log:       c.Log,
	}

	c.Log.Debugf("Parsed image: ID=%s, Name=%s, Tag=%s", id, name, tag)
	return image
}

// SystemStart starts the Apple Container system services
func (c *AppleContainerCommand) SystemStart() error {
	c.Log.Info("Starting Apple Container system services")
	return c.OSCommand.RunCommand("container system start")
}

// SystemStop stops the Apple Container system services
func (c *AppleContainerCommand) SystemStop() error {
	c.Log.Info("Stopping Apple Container system services")
	return c.OSCommand.RunCommand("container system stop")
}

// SystemStatus gets the status of Apple Container system services
func (c *AppleContainerCommand) SystemStatus() (map[string]interface{}, error) {
	c.Log.Info("Getting Apple Container system status")

	output, err := c.OSCommand.RunCommandWithOutput("container system status --format json")
	if err != nil {
		return nil, fmt.Errorf("failed to get system status: %w", err)
	}

	var status map[string]interface{}
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		return nil, fmt.Errorf("failed to parse system status: %w", err)
	}

	return status, nil
}
