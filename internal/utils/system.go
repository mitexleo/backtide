package utils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// CheckRootPrivileges checks if the program is running with root privileges
func CheckRootPrivileges() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("this operation requires root privileges")
	}
	return nil
}

// DirectoryExists checks if a directory exists
func DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// CreateDirectory creates a directory with proper permissions
func CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// RemoveDirectory removes a directory and all its contents
func RemoveDirectory(path string) error {
	return os.RemoveAll(path)
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetDirectorySize returns the total size of a directory in bytes
func GetDirectorySize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

// CopyFile copies a file from source to destination
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// ExecuteCommand executes a shell command and returns output
func ExecuteCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ExecuteCommandWithEnv executes a shell command with environment variables
func ExecuteCommandWithEnv(env []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// IsCommandAvailable checks if a command is available in the system PATH
func IsCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// GetFileUID returns the UID of a file
func GetFileUID(path string) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("failed to get file stat")
	}

	return int(stat.Uid), nil
}

// GetFileGID returns the GID of a file
func GetFileGID(path string) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("failed to get file stat")
	}

	return int(stat.Gid), nil
}

// GetFileMode returns the file mode as string
func GetFileMode(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	return info.Mode().String(), nil
}

// SetFilePermissions sets file permissions and ownership
func SetFilePermissions(path string, mode os.FileMode, uid, gid int) error {
	if err := os.Chmod(path, mode); err != nil {
		return err
	}
	return os.Chown(path, uid, gid)
}

// ParseFileMode parses a file mode string to os.FileMode
func ParseFileMode(modeStr string) (os.FileMode, error) {
	// Remove leading characters to get just the permission bits
	if len(modeStr) > 9 {
		modeStr = modeStr[len(modeStr)-9:]
	}

	var mode os.FileMode
	for i, c := range modeStr {
		switch c {
		case 'r':
			mode |= 1 << (8 - uint(i))
		case 'w':
			mode |= 1 << (7 - uint(i))
		case 'x':
			mode |= 1 << (6 - uint(i))
		case '-':
			// Skip, already 0
		default:
			return 0, fmt.Errorf("invalid file mode character: %c", c)
		}
	}

	return mode, nil
}

// CleanPath cleans and normalizes a file path
func CleanPath(path string) string {
	return filepath.Clean(path)
}

// JoinPaths joins multiple path elements
func JoinPaths(elem ...string) string {
	return filepath.Join(elem...)
}

// GetHomeDir returns the user's home directory
func GetHomeDir() (string, error) {
	return os.UserHomeDir()
}

// GetCurrentDir returns the current working directory
func GetCurrentDir() (string, error) {
	return os.Getwd()
}

// ChangeDir changes the current working directory
func ChangeDir(path string) error {
	return os.Chdir(path)
}

// IsEmptyDir checks if a directory is empty
func IsEmptyDir(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// EnsureTrailingSlash ensures a path ends with a trailing slash
func EnsureTrailingSlash(path string) string {
	if !strings.HasSuffix(path, string(filepath.Separator)) {
		return path + string(filepath.Separator)
	}
	return path
}

// RemoveTrailingSlash removes trailing slash from a path
func RemoveTrailingSlash(path string) string {
	return strings.TrimSuffix(path, string(filepath.Separator))
}
