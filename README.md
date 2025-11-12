# Backtide - Docker Backup Utility

A comprehensive backup utility for Docker-based applications with S3 integration, built in Go with Cobra CLI framework.

## Features

- **Multi-directory Backup**: Backup multiple directories with compression
- **Docker Container Management**: Stop and restart containers during backup operations
- **S3FS Integration**: Automatic S3FS installation and S3 bucket mounting
- **Permission Preservation**: Store and restore file permissions and ownership
- **Retention Policies**: Configurable backup retention (days, count, monthly)
- **Scheduling Options**: Both systemd timer and cron job support
- **Metadata Tracking**: Comprehensive backup metadata and checksums
- **No Database Required**: File-based state management

## Installation

### Prerequisites

- Go 1.25 or later
- Docker (optional, for container management)
- Root privileges (for S3FS operations)

### Build from Source

```bash
git clone https://github.com/mitexleo/backtide.git
cd backtide
go build -o backtide
sudo mv backtide /usr/local/bin/
```

## Quick Start

1. **Initialize configuration**:
   ```bash
   backtide init
   ```

2. **Edit configuration**:
   ```bash
   nano ~/.backtide.yaml
   ```

3. **Test backup**:
   ```bash
   backtide backup --dry-run
   ```

4. **Set up automated backups**:
   ```bash
   sudo backtide systemd install
   ```

## Configuration

### Example Configuration

```yaml
# ~/.backtide.yaml
backup_path: /mnt/backup
temp_path: /tmp/backtide

directories:
  - path: /var/lib/docker/volumes
    name: docker-volumes
    compression: true
  - path: /opt/myapp/data
    name: app-data
    compression: true

s3:
  bucket: my-backup-bucket
  region: us-east-1
  access_key: YOUR_ACCESS_KEY
  secret_key: YOUR_SECRET_KEY
  mount_point: /mnt/s3backup
  use_path_style: false

retention:
  keep_days: 30
  keep_count: 10
  keep_monthly: 6
```

### Configuration Options

- **directories**: List of directories to backup
  - `path`: Source directory path
  - `name`: Backup identifier name
  - `compression`: Enable/disable compression

- **s3**: S3 bucket configuration
  - `bucket`: S3 bucket name
  - `region`: AWS region
  - `access_key`: AWS access key
  - `secret_key`: AWS secret key
  - `mount_point`: Local mount point
  - `use_path_style`: Use path-style S3 URLs

- **retention**: Backup retention policy
  - `keep_days`: Keep backups for X days
  - `keep_count`: Keep last X backups
  - `keep_monthly`: Keep X monthly backups

## Usage

### Backup Operations

```bash
# Create a backup
backtide backup

# Create backup without Docker operations
backtide backup --skip-docker

# Create backup without S3 operations
backtide backup --skip-s3

# Dry run to see what would happen
backtide backup --dry-run
```

### Restore Operations

```bash
# Restore latest backup
backtide restore

# Restore specific backup
backtide restore backup-2024-01-15-10-30-00

# Force restore without confirmation
backtide restore --force
```

### Backup Management

```bash
# List all backups
backtide list

# List with detailed information
backtide list --detailed

# Clean up old backups
backtide cleanup

# Cleanup dry run
backtide cleanup --dry-run
```

### Scheduling

#### Systemd Timer (Recommended)

```bash
# Install systemd service and timer
sudo backtide systemd install

# Check status
backtide systemd status

# Uninstall
sudo backtide systemd uninstall
```

#### Cron Jobs

```bash
# Install cron job (daily at 2 AM)
backtide cron install

# Install with custom schedule
backtide cron install --schedule "0 3 * * *"

# Check cron status
backtide cron status

# Uninstall cron job
backtide cron uninstall
```

## How It Works

### Backup Process

1. **Container Management**: Stops all running Docker containers (optional)
2. **S3 Setup**: Installs and mounts S3 bucket using s3fs (optional)
3. **Backup Creation**: Creates compressed tar archives of specified directories
4. **Metadata Storage**: Saves file permissions, checksums, and backup metadata
5. **Container Restoration**: Restarts previously stopped containers
6. **Cleanup**: Applies retention policies to remove old backups

### File Structure

```
/var/lib/backtide/
├── containers.json      # Docker container state
└── backups/
    └── backup-2024-01-15-10-30-00/
        ├── metadata.json
        ├── docker-volumes.tar.gz
        └── app-data.tar.gz
```

### Backup Metadata

Each backup includes comprehensive metadata:
- Backup ID and timestamp
- Directory sizes and file counts
- File permissions and ownership
- SHA-256 checksums
- Compression status

## Security Considerations

- **Credentials**: S3 credentials are stored in `/etc/passwd-s3fs` with 600 permissions
- **Root Privileges**: S3FS operations require root privileges
- **File Permissions**: Original file permissions are preserved during backup/restore

## Troubleshooting

### Common Issues

1. **S3FS Installation Fails**
   - Ensure you have root privileges
   - Check internet connectivity
   - Verify package manager availability

2. **Docker Containers Not Stopping**
   - Ensure Docker is running
   - Check if user has Docker permissions
   - Use `--skip-docker` to bypass container management

3. **Backup Fails Due to Permission Issues**
   - Run with appropriate privileges
   - Check directory permissions
   - Use `--verbose` for detailed error messages

### Debug Mode

```bash
# Enable verbose output
backtide --verbose backup

# Check system requirements
docker info
systemctl status docker
```

## Development

### Project Structure

```
backtide/
├── cmd/                 # CLI commands
│   ├── root.go
│   ├── backup.go
│   ├── restore.go
│   ├── list.go
│   ├── cleanup.go
│   ├── systemd.go
│   ├── cron.go
│   └── init.go
├── internal/            # Internal packages
│   ├── backup/         # Backup management
│   ├── docker/         # Docker operations
│   ├── s3fs/          # S3FS management
│   ├── config/         # Configuration handling
│   └── utils/          # Utility functions
├── main.go
└── README.md
```

### Building and Testing

```bash
# Build
go build -o backtide

# Test
go test ./...

# Run with specific config
backtide --config /path/to/config.yaml backup
```

## License

MIT License - see LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## Support

For issues and feature requests, please create an issue on the GitHub repository.