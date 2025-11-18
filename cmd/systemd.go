package cmd

import (
	"fmt"
	"os"

	"github.com/mitexleo/backtide/internal/commands"
	"github.com/mitexleo/backtide/internal/systemd"
	"github.com/spf13/cobra"
)

// systemdCmd represents the systemd command (kept for backward compatibility)
// This command is deprecated and will be removed in future versions
var systemdCmd = &cobra.Command{
	Use:    "systemd",
	Short:  "[DEPRECATED] Systemd service management is now automatic",
	Long:   `[DEPRECATED] Systemd service management is now handled automatically during updates and initialization.`,
	Hidden: true, // Hide from help since it's deprecated
}

func init() {
	// Register with command registry (but keep it hidden)
	commands.RegisterCommand("systemd", systemdCmd)
}

// ensureSystemdService ensures the systemd service is properly configured
// This is called automatically during init and update operations
func ensureSystemdService(configPath string) error {
	// Only update systemd service when running as root
	if os.Geteuid() != 0 {
		return nil // Skip if not root
	}

	// Create systemd service manager
	manager := systemd.NewServiceManager("backtide", "", configPath, "root")

	// Check if service directory exists
	systemdDir := "/etc/systemd/system"
	if _, err := os.Stat(systemdDir); os.IsNotExist(err) {
		// Systemd directory doesn't exist, skip
		return nil
	}

	// Always update service file to latest version
	if err := manager.UpdateServiceFile(); err != nil {
		return fmt.Errorf("failed to update systemd service: %w", err)
	}

	// Check if service is currently enabled
	if installed, _ := manager.IsServiceInstalled(); installed {
		// Service exists, ensure it's using the latest version
		fmt.Println("🔄 Updating systemd service to latest version...")

		// Restart service to pick up changes
		if err := manager.StopService(); err != nil {
			fmt.Printf("⚠️  Warning: Could not stop service: %v\n", err)
		}

		if err := manager.EnableService(); err != nil {
			fmt.Printf("⚠️  Warning: Could not enable service: %v\n", err)
		}

		if err := manager.StartService(); err != nil {
			fmt.Printf("⚠️  Warning: Could not start service: %v\n", err)
		}

		fmt.Println("✅ Systemd service updated successfully")
	}

	return nil
}

// removeSystemdService removes the systemd service (used during uninstall)
func removeSystemdService() error {
	// Only remove systemd service when running as root
	if os.Geteuid() != 0 {
		return nil
	}

	manager := systemd.NewServiceManager("backtide", "", "", "")

	// Stop and disable service
	if err := manager.StopService(); err != nil {
		fmt.Printf("⚠️  Warning: Failed to stop service: %v\n", err)
	}

	if err := manager.DisableService(); err != nil {
		fmt.Printf("⚠️  Warning: Failed to disable service: %v\n", err)
	}

	// Remove service file
	serviceFile := manager.GetServiceFilePath()
	if err := os.Remove(serviceFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// Remove timer file if it exists (clean up old approach)
	timerFile := manager.GetTimerFilePath()
	if _, err := os.Stat(timerFile); err == nil {
		os.Remove(timerFile)
	}

	// Reload systemd
	if err := manager.ReloadDaemon(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	return nil
}
