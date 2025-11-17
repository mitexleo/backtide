package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ServiceManager provides abstraction for systemd service operations
type ServiceManager struct {
	ServiceName string
	BinaryPath  string
	ConfigPath  string
	User        string
}

// NewServiceManager creates a new systemd service manager
func NewServiceManager(serviceName, binaryPath, configPath, user string) *ServiceManager {
	return &ServiceManager{
		ServiceName: serviceName,
		BinaryPath:  binaryPath,
		ConfigPath:  configPath,
		User:        user,
	}
}

// ServiceInfo represents information about a systemd service
type ServiceInfo struct {
	Name        string
	IsInstalled bool
	IsRunning   bool
	IsEnabled   bool
	BinaryPath  string
}

// ServiceStatus represents the current status of a systemd service
type ServiceStatus struct {
	ServiceName string
	IsActive    bool
	IsEnabled   bool
	IsRunning   bool
	LoadState   string
	ActiveState string
	SubState    string
}

// IsServiceInstalled checks if the systemd service is installed
func (sm *ServiceManager) IsServiceInstalled() (bool, error) {
	// Check if service file exists
	serviceFile := sm.GetServiceFilePath()
	if _, err := os.Stat(serviceFile); err == nil {
		return true, nil
	}

	// Also check via systemctl as fallback
	cmd := exec.Command("systemctl", "list-unit-files", sm.ServiceName+".service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to check service installation: %v", err)
	}

	return strings.Contains(string(output), sm.ServiceName+".service"), nil
}

// GetServiceStatus retrieves detailed status of the systemd service
func (sm *ServiceManager) GetServiceStatus() (*ServiceStatus, error) {
	// First check if service is installed
	isInstalled, err := sm.IsServiceInstalled()
	if err != nil {
		return nil, err
	}
	if !isInstalled {
		return &ServiceStatus{
			ServiceName: sm.ServiceName,
			LoadState:   "not-found",
			ActiveState: "inactive",
			SubState:    "dead",
			IsEnabled:   false,
			IsActive:    false,
			IsRunning:   false,
		}, nil
	}

	cmd := exec.Command("systemctl", "show", sm.ServiceName+".service", "--property=LoadState,ActiveState,SubState")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get service status: %v", err)
	}

	status := &ServiceStatus{ServiceName: sm.ServiceName}
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		switch parts[0] {
		case "LoadState":
			status.LoadState = parts[1]
			status.IsEnabled = parts[1] == "loaded"
		case "ActiveState":
			status.ActiveState = parts[1]
			status.IsActive = parts[1] == "active"
		case "SubState":
			status.SubState = parts[1]
			status.IsRunning = parts[1] == "running"
		}
	}

	return status, nil
}

// GenerateServiceFile generates the systemd service file content
func (sm *ServiceManager) GenerateServiceFile() string {
	return `[Unit]
Description=Backtide Backup Service
Documentation=https://github.com/mitexleo/backtide
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=` + sm.User + `
ExecStart=backtide daemon
StandardOutput=journal
StandardError=journal
Restart=always
RestartSec=10
TimeoutStopSec=30

[Install]
WantedBy=multi-user.target
`
}

// GenerateTimerFile generates the systemd timer file content
// DEPRECATED: Backtide now uses continuous daemon for scheduling
func (sm *ServiceManager) GenerateTimerFile(schedule string) string {
	return ""
}

// ReloadDaemon reloads the systemd daemon
func (sm *ServiceManager) ReloadDaemon() error {
	cmd := exec.Command("systemctl", "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %s, error: %v", string(output), err)
	}
	return nil
}

// EnableService enables the systemd service
func (sm *ServiceManager) EnableService() error {
	cmd := exec.Command("systemctl", "enable", sm.ServiceName+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable service: %s, error: %v", string(output), err)
	}
	return nil
}

// StartService starts the systemd service
func (sm *ServiceManager) StartService() error {
	cmd := exec.Command("systemctl", "start", sm.ServiceName+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start service: %s, error: %v", string(output), err)
	}
	return nil
}

// StopService stops the systemd service
func (sm *ServiceManager) StopService() error {
	cmd := exec.Command("systemctl", "stop", sm.ServiceName+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop service: %s, error: %v", string(output), err)
	}
	return nil
}

// DisableService disables the systemd service
func (sm *ServiceManager) DisableService() error {
	cmd := exec.Command("systemctl", "disable", sm.ServiceName+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to disable service: %s, error: %v", string(output), err)
	}
	return nil
}

// GetServiceFilePath returns the full path to the service file
func (sm *ServiceManager) GetServiceFilePath() string {
	return filepath.Join("/etc/systemd/system", sm.ServiceName+".service")
}

// GetTimerFilePath returns the full path to the timer file
func (sm *ServiceManager) GetTimerFilePath() string {
	return filepath.Join("/etc/systemd/system", sm.ServiceName+".timer")
}

// UpdateServiceFile updates the systemd service file for continuous daemon
func (sm *ServiceManager) UpdateServiceFile() error {
	// Check if service file already exists
	serviceFile := sm.GetServiceFilePath()
	serviceExists := false

	if _, err := os.Stat(serviceFile); err == nil {
		serviceExists = true
	}

	// Create service file for continuous daemon
	serviceContent := sm.GenerateServiceFile()
	if err := os.WriteFile(serviceFile, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to update service file: %v", err)
	}

	// Remove any existing timer file (clean up old approach)
	timerFile := sm.GetTimerFilePath()
	if _, err := os.Stat(timerFile); err == nil {
		os.Remove(timerFile)
	}

	// Reload systemd daemon
	if err := sm.ReloadDaemon(); err != nil {
		return fmt.Errorf("failed to reload systemd after update: %v", err)
	}

	// Provide feedback about what was done
	if serviceExists {
		fmt.Printf("  🔄 Updated existing service file\n")
	} else {
		fmt.Printf("  📝 Created new service file\n")
	}

	return nil
}
