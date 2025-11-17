# Backtide System Configuration Architecture

## Overview

Backtide now uses a centralized system-wide configuration architecture located in `/etc/backtide/`. This document explains the architecture, file locations, and configuration management for system administrators and developers.

## Configuration Locations

### Primary Configuration Directory
```
/etc/backtide/
├── config.toml          # Main configuration file
└── s3-credentials/      # S3 bucket credentials (secure)
    ├── passwd-s3fs-bucket-id-1
    ├── passwd-s3fs-bucket-id-2
    └── ...
```

### File Permissions
- **Configuration files**: `0644` (readable by all, writable by owner)
- **Credential files**: `0600` (read/write by owner only)
- **Directories**: `0755` for config, `0700` for credentials

## Architecture Design

### 1. Configuration Loading Priority

The system searches for configuration files in this order:

1. `/etc/backtide/config.toml` (Primary - system-wide)
2. `/etc/backtide/backtide.toml` (Alternative system location)
3. Development locations (fallback for testing)

**Code Implementation:**
```go
// FindConfigFile searches for configuration file in common locations
func FindConfigFile() string {
    locations := []string{
        "/etc/backtide/config.toml",
        "/etc/backtide/backtide.toml",
    }
    // ... fallback to development locations
}
```

### 2. Credential Management

Each S3 bucket gets its own secure credential file:

- **File naming**: `passwd-s3fs-{bucket-id}`
- **Format**: `access_key:secret_key`
- **Location**: `/etc/backtide/s3-credentials/`
- **Security**: Files are created with `0600` permissions

**Code Implementation:**
```go
// getCredentialsFilePath returns the path to credentials file
func getCredentialsFilePath(bucketID string) string {
    return filepath.Join("/etc", "backtide", "s3-credentials", 
        fmt.Sprintf("passwd-s3fs-%s", bucketID))
}
```

### 3. Automatic Directory Creation

The system automatically creates necessary directories:

```go
// EnsureSystemDirectories creates necessary system directories
func EnsureSystemDirectories() error {
    // Create /etc/backtide directory for configuration
    os.MkdirAll("/etc/backtide", 0755)
    
    // Create /etc/backtide/s3-credentials directory for credentials  
    credsDir := filepath.Join("/etc", "backtide", "s3-credentials")
    os.MkdirAll(credsDir, 0700)
    
    return nil
}
```

## Command Behavior

### S3 Add Command Flow
1. **Check dependencies** - Verify s3fs is installed
2. **Ensure directories** - Create `/etc/backtide/` if needed
3. **Configure bucket** - Interactive bucket setup
4. **Save configuration** - Update `/etc/backtide/config.toml`
5. **Create mount point** - System directory creation
6. **Add to fstab** - Persistent mount configuration
7. **Store credentials** - Secure credential file creation

### S3 Remove Command Flow  
1. **Remove configuration** - Delete from `/etc/backtide/config.toml`
2. **Clean credentials** - Remove credential file
3. **Remove fstab entry** - Delete from `/etc/fstab`
4. **Clean mount point** - Remove empty directories

### S3 Test Command Flow
1. **Verify s3fs** - Check dependency availability
2. **Setup credentials** - Create temporary credential file
3. **Mount test** - Attempt S3 bucket mounting
4. **File operations** - Read/write/delete test files
5. **Cleanup** - Remove test credentials and unmount

## Sudo Requirements

### Commands Requiring Sudo
- `backtide init` - Writing to `/etc/backtide/`
- `backtide s3 add` - Writing to system directories and fstab
- `backtide s3 remove` - Removing from system directories and fstab
- `backtide s3 test` - System directory operations

### Automatic Sudo Detection
The system detects when sudo is needed and provides clear instructions:

```go
if err := s3fsManager.AddToFstab(); err != nil {
    fmt.Printf("⚠️  Warning: Could not add to /etc/fstab: %v\n", err)
    fmt.Println("   You may need to run with sudo for system configuration")
    fmt.Println("   Try: sudo backtide s3 add")
}
```

## Configuration File Structure

### Main Configuration (`/etc/backtide/config.toml`)
```toml
backup_path = "/var/lib/backtide"
temp_path = "/tmp/backtide"

[[buckets]]
id = "bucket-1234567890"
name = "Production Backup"
bucket = "my-backup-bucket"
region = "us-east-1"
access_key = "AKIA..."  # Masked in display
secret_key = "wJal..."  # Masked in display
endpoint = ""
mount_point = "/mnt/s3backup"
use_path_style = false
provider = "AWS S3"

[[jobs]]
id = "job-default"
name = "Default Backup"
bucket_id = "bucket-1234567890"
# ... job configuration
```

### Credential File Format
```
AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

## Security Considerations

### 1. Credential Isolation
- Each bucket has separate credential files
- No credential sharing between buckets
- Individual file permissions (`0600`)

### 2. System Access Control
- Configuration readable by all users
- Credentials accessible only to root/authorized users
- Clear separation between config and sensitive data

### 3. Backup and Recovery
- Centralized configuration simplifies backup
- All critical data in `/etc/backtide/`
- Easy to migrate between systems

## Migration from User Configuration

### From User Home Directory
If migrating from user-based configuration:

1. **Stop Backtide services**
2. **Copy configuration**:
   ```bash
   sudo cp ~/.backtide.toml /etc/backtide/config.toml
   ```
3. **Copy credentials**:
   ```bash
   sudo cp -r ~/.config/backtide/s3-credentials/ /etc/backtide/
   sudo chmod 600 /etc/backtide/s3-credentials/*
   ```
4. **Update fstab entries** to use new credential paths
5. **Remove old configuration**:
   ```bash
   rm ~/.backtide.toml
   rm -rf ~/.config/backtide/
   ```

## Troubleshooting

### Common Issues

1. **Permission Denied**
   ```
   ❌ Error: permission denied
   💡 Solution: Run with sudo for system operations
   ```

2. **Configuration Not Found**
   ```
   ⚠️ Using development configuration
   💡 For production, use: /etc/backtide/config.toml
   ```

3. **Credential File Issues**
   ```bash
   # Check credential file permissions
   sudo ls -la /etc/backtide/s3-credentials/
   
   # Verify credential file content
   sudo cat /etc/backtide/s3-credentials/passwd-s3fs-bucket-id
   ```

### Debugging Commands

```bash
# Check configuration file
sudo cat /etc/backtide/config.toml

# List all buckets
sudo backtide s3 list

# Test specific bucket
sudo backtide s3 test bucket-id

# Verify fstab entries
cat /etc/fstab | grep s3fs

# Check mount points
mount | grep s3fs
```

## Best Practices

### 1. Regular Backups
```bash
# Backup configuration
sudo tar -czf backtide-config-backup.tar.gz /etc/backtide/
```

### 2. Security Audits
```bash
# Check file permissions
sudo find /etc/backtide -type f -exec ls -la {} \;

# Verify no world-writable files
sudo find /etc/backtide -perm -o+w
```

### 3. Monitoring
- Monitor `/var/log/backtide/` for operation logs
- Check systemd service status for scheduled backups
- Regular verification of S3 connectivity

This architecture provides a secure, maintainable, and system-administrator-friendly configuration management system for Backtide.