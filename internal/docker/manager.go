package docker

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	if len(containers) == 0 {
		fmt.Println("No running containers found to stop")
		return []config.DockerContainerInfo{}, nil
	}

	fmt.Printf("Found %d running containers\n", len(containers))

	var stoppedContainers []config.DockerContainerInfo
	var failedContainers []string
	currentTime := time.Now()

	for _, container := range containers {
		fmt.Printf("Attempting to stop container: %s (%s) - Status: %s\n",
			container.Name, container.ID[:12], container.Status)

		// Stop the container
		cmd := exec.Command("docker", "stop", container.ID)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: Failed to stop container %s: %v\n", container.Name, err)
			failedContainers = append(failedContainers, container.Name)
			continue
		}

		// Update container status and timestamp
		container.Status = "stopped"
		container.Stopped = currentTime
		stoppedContainers = append(stoppedContainers, container)

		fmt.Printf("✅ Successfully stopped container: %s (%s)\n", container.Name, container.ID[:12])
	}

	// Save stopped containers to state file even if some failed
	if len(stoppedContainers) > 0 {
		if err := dm.saveStoppedContainers(stoppedContainers); err != nil {
			return stoppedContainers, fmt.Errorf("failed to save container state: %w", err)
		}
	}

	// Report results
	if len(failedContainers) > 0 {
		fmt.Printf("Warning: Failed to stop %d containers: %s\n",
			len(failedContainers), strings.Join(failedContainers, ", "))
	}

	fmt.Printf("✅ Successfully stopped %d out of %d containers\n",
		len(stoppedContainers), len(containers))

	return stoppedContainers, nil
}

// RestoreContainers restores previously stopped containers
func (dm *DockerManager) RestoreContainers() error {
	stoppedContainers, err := dm.loadStoppedContainers()
	if err != nil {
		return fmt.Errorf("failed to load container state: %w", err)
	}

	if len(stoppedContainers) == 0 {
		fmt.Println("No containers to restore")
		return nil
	}

	fmt.Printf("Attempting to restore %d containers\n", len(stoppedContainers))

	var restoredCount int
	var failedContainers []string

	for _, container := range stoppedContainers {
		fmt.Printf("Attempting to start container: %s (%s)\n", container.Name, container.ID[:12])

		// Start the container
		cmd := exec.Command("docker", "start", container.ID)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: Failed to start container %s: %v\n", container.Name, err)
			failedContainers = append(failedContainers, container.Name)
			continue
		}

		fmt.Printf("✅ Successfully restarted container: %s (%s)\n", container.Name, container.ID[:12])
		restoredCount++
	}

	// Clear the state file after restoration attempt
	if err := dm.clearStoppedContainers(); err != nil {
		fmt.Printf("Warning: Failed to clear container state: %v\n", err)
	}

	// Report results
	if len(failedContainers) > 0 {
		return fmt.Errorf("failed to restart %d containers: %s",
			len(failedContainers), strings.Join(failedContainers, ", "))
	}

	fmt.Printf("✅ Successfully restored %d containers\n", restoredCount)
	return nil
}

// GetStoppedContainers returns the list of currently stopped containers
func (dm *DockerManager) GetStoppedContainers() ([]config.DockerContainerInfo, error) {
	return dm.loadStoppedContainers()
}

// GetRunningContainers returns the list of currently running containers (for testing)
func (dm *DockerManager) GetRunningContainers() ([]config.DockerContainerInfo, error) {
	return dm.getRunningContainers()
}

// getRunningContainers retrieves all containers that should be stopped for backup
func (dm *DockerManager) getRunningContainers() ([]config.DockerContainerInfo, error) {
	// Use docker ps without status filter to get all containers that are not stopped/exited
	// This includes running, restarting, paused, and other active states
	cmd := exec.Command("docker", "ps", "--format", "{{.ID}}|{{.Names}}|{{.Image}}|{{.Status}}")

	output, err := cmd.Output()
	if err != nil {
		// Check if Docker is available
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "permission denied") {
				return nil, fmt.Errorf("docker permission denied - try running with sudo or add user to docker group")
			}
			if strings.Contains(stderr, "Cannot connect") {
				return nil, fmt.Errorf("docker daemon not running - start docker service first")
			}
		}
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var containers []config.DockerContainerInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) != 4 {
			fmt.Printf("Warning: Skipping malformed container line: %s\n", line)
			continue
		}

		container := config.DockerContainerInfo{
			ID:     strings.TrimSpace(parts[0]),
			Name:   strings.TrimSpace(parts[1]),
			Image:  strings.TrimSpace(parts[2]),
			Status: strings.TrimSpace(parts[3]),
		}

		// Skip containers that are already stopped or exited
		if strings.Contains(strings.ToLower(container.Status), "exited") {
			continue
		}

		containers = append(containers, container)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning container output: %w", err)
	}

	return containers, nil
}

// saveStoppedContainers saves stopped containers to the state file
func (dm *DockerManager) saveStoppedContainers(containers []config.DockerContainerInfo) error {
	data, err := json.MarshalIndent(containers, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal container data: %w", err)
	}

	// Ensure directory exists
	dir := dm.getStateFileDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Write to temporary file first, then rename for atomic operation
	tempFile := dm.stateFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary state file: %w", err)
	}

	if err := os.Rename(tempFile, dm.stateFile); err != nil {
		// Clean up temp file if rename fails
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename state file: %w", err)
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

// getStateFileDir returns the directory containing the state file
func (dm *DockerManager) getStateFileDir() string {
	return filepath.Dir(dm.stateFile)
}

// CheckDockerAvailable checks if Docker is available and running
func (dm *DockerManager) CheckDockerAvailable() error {
	cmd := exec.Command("docker", "info")
	output, err := cmd.CombinedOutput()
	if err != nil {
		errorMsg := string(output)
		if strings.Contains(errorMsg, "permission denied") {
			return fmt.Errorf("docker permission denied - try running with sudo or add user to docker group")
		}
		if strings.Contains(errorMsg, "Cannot connect") {
			return fmt.Errorf("docker daemon not running - start docker service first")
		}
		return fmt.Errorf("docker is not available: %w - output: %s", err, errorMsg)
	}
	return nil
}
