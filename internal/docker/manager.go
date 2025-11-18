package docker

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mitexleo/backtide/internal/config"
)

// DockerManager handles Docker container operations
type DockerManager struct {
	stateFile string
}

// NewDockerManager creates a new Docker manager instance
func NewDockerManager(stateFile string) *DockerManager {
	return &DockerManager{
		stateFile: stateFile,
	}
}

// StopContainers stops all running Docker containers and returns their info
func (dm *DockerManager) StopContainers() ([]config.DockerContainerInfo, error) {
	containers, err := dm.getRunningContainers()
	if err != nil {
		return nil, fmt.Errorf("failed to get running containers: %w", err)
	}

	var stoppedContainers []config.DockerContainerInfo
	currentTime := time.Now()

	for _, container := range containers {
		// Skip containers that are already stopped
		if container.Status != "running" {
			continue
		}

		// Stop the container
		// Stop the container
		cmd := exec.Command("docker", "stop", container.ID)
		if err := cmd.Run(); err != nil {
			return stoppedContainers, fmt.Errorf("failed to stop container %s: %w", container.Name, err)
		}

		// Update container status and timestamp
		container.Status = "stopped"
		container.Stopped = currentTime
		stoppedContainers = append(stoppedContainers, container)

		fmt.Printf("Stopped container: %s (%s)\n", container.Name, container.ID[:12])
	}

	// Save stopped containers to state file
	if err := dm.saveStoppedContainers(stoppedContainers); err != nil {
		return stoppedContainers, fmt.Errorf("failed to save container state: %w", err)
	}

	return stoppedContainers, nil
}

// RestoreContainers restores previously stopped containers
func (dm *DockerManager) RestoreContainers() error {
	stoppedContainers, err := dm.loadStoppedContainers()
	if err != nil {
		return fmt.Errorf("failed to load container state: %w", err)
	}

	var restoredCount int
	var errors []string

	for _, container := range stoppedContainers {
		// Start the container
		// Start the container
		cmd := exec.Command("docker", "start", container.ID)
		if err := cmd.Run(); err != nil {
			errors = append(errors, fmt.Sprintf("failed to start container %s: %v", container.Name, err))
			continue
		}

		fmt.Printf("Restarted container: %s (%s)\n", container.Name, container.ID[:12])
		restoredCount++
	}

	// Clear the state file after successful restoration
	if err := dm.clearStoppedContainers(); err != nil {
		errors = append(errors, fmt.Sprintf("failed to clear container state: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("some containers failed to restart: %s", strings.Join(errors, "; "))
	}

	fmt.Printf("Successfully restored %d containers\n", restoredCount)
	return nil
}

// GetStoppedContainers returns the list of currently stopped containers
func (dm *DockerManager) GetStoppedContainers() ([]config.DockerContainerInfo, error) {
	return dm.loadStoppedContainers()
}

// getRunningContainers retrieves all running Docker containers
func (dm *DockerManager) getRunningContainers() ([]config.DockerContainerInfo, error) {
	cmd := exec.Command("docker", "ps", "--format", "{{.ID}}|{{.Names}}|{{.Image}}|{{.Status}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Debug: Log raw output to understand what containers are found
	fmt.Printf("DEBUG: Docker ps raw output: %s\n", string(output))

	var containers []config.DockerContainerInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) != 4 {
			fmt.Printf("DEBUG: Skipping malformed line: %s\n", line)
			continue
		}

		container := config.DockerContainerInfo{
			ID:     strings.TrimSpace(parts[0]),
			Name:   strings.TrimSpace(parts[1]),
			Image:  strings.TrimSpace(parts[2]),
			Status: strings.TrimSpace(parts[3]),
		}
		containers = append(containers, container)
		fmt.Printf("DEBUG: Found container: %s (%s) - %s\n", container.Name, container.ID[:12], container.Status)
	}

	fmt.Printf("DEBUG: Total containers found: %d\n", len(containers))

	return containers, nil
}

// saveStoppedContainers saves stopped containers to the state file
func (dm *DockerManager) saveStoppedContainers(containers []config.DockerContainerInfo) error {
	data, err := json.MarshalIndent(containers, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal container data: %w", err)
	}

	if err := os.WriteFile(dm.stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// loadStoppedContainers loads stopped containers from the state file
func (dm *DockerManager) loadStoppedContainers() ([]config.DockerContainerInfo, error) {
	data, err := os.ReadFile(dm.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []config.DockerContainerInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var containers []config.DockerContainerInfo
	if err := json.Unmarshal(data, &containers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal container data: %w", err)
	}

	return containers, nil
}

// clearStoppedContainers clears the stopped containers state file
func (dm *DockerManager) clearStoppedContainers() error {
	if err := os.Remove(dm.stateFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}
	return nil
}

// CheckDockerAvailable checks if Docker is available and running
func (dm *DockerManager) CheckDockerAvailable() error {
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker is not available or not running: %w", err)
	}
	return nil
}
