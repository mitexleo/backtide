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
   This will guide you through interactive setup for backup jobs, including:
   - Job naming and description
   - Backup scheduling (daily, weekly, monthly, custom cron)
   - Retention policies
   - S3 storage configuration
   - Docker container management
   - Directory selection

2. **Test backup**:
   ```bash
   # Test specific job
   backtide backup my-backup-job --dry-run
   
   # Test all enabled jobs
   backtide backup --all --dry-run
   
   # Test default job
   backtide backup --dry-run
   ```

3. **Set up automated backups**:
   ```bash
   # For single job system
   sudo backtide systemd install
   
   # For multiple jobs with different schedules
   sudo backtide systemd-jobs install
   ```

*Alternative: Manual configuration*
```bash
backtide init --skip-interactive
nano ~/.backtide.yaml  # Edit manually
```

## Job-Based Backup System

Backtide now supports multiple backup jobs with different configurations:

- **Multiple Jobs**: Run different backups for different purposes
- **Independent Scheduling**: Each job can have its own schedule
- **Flexible Configuration**: Different S3 buckets, directories, and retention per job
- **Granular Control**: Enable/disable jobs individually

### Example Multi-Job Configuration:
```yaml
jobs:
  - name: daily-docker-backup
    description: Daily backup of Docker volumes
    enabled: true
    schedule:
      type: systemd
      interval: daily
      enabled: true
    directories:
      - path: /var/lib/docker/volumes
        name: docker-volumes
        compression: true
    retention:
      keep_days: 7
      keep_count: 7
      keep_monthly: 0
    skip_docker: false
    skip_s3: false
  
  - name: weekly-app-backup  
    description: Weekly backup of application data
    enabled: true
    schedule:
      type: systemd
      interval: weekly
      enabled: true
    directories:
      - path: /opt/myapp/data
        name: app-data
        compression: true
    retention:
      keep_days: 30
      keep_count: 4
      keep_monthly: 6
    skip_docker: true
    skip_s3: false
```

## Job Management

### Managing Backup Jobs

```bash
# List all jobs
backtide jobs list

# List with detailed information
backtide jobs list --detailed

# Show job status and next run times
backtide jobs status

# Enable/disable specific jobs
backtide jobs enable daily-docker-backup
backtide jobs disable weekly-app-backup

# Run specific job
backtide backup daily-docker-backup

# Run all enabled jobs
backtide backup --all
```

### Job Status Information

Each job shows:
- ✅ Enabled/❌ Disabled status
- Schedule information
- Next scheduled run time
- Directory count and configuration
- Retention policy details
- Docker and S3 settings

## Configuration

### Example Single Job Configuration

```yaml
# ~/.backtide.yaml
backup_path: /mnt/backup
temp_path: /tmp/backtide

jobs:
  - name: default-backup
    description: Default backup job
    enabled: true
    schedule:
      type: systemd
      interval: daily
      enabled: true
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
      endpoint: ""
      mount_point: /mnt/s3backup
      use_path_style: false
    retention:
      keep_days: 30
      keep_count: 10
      keep_monthly: 6
    skip_docker: false
    skip_s3: false
```

### Job Configuration Options

Each backup job supports:

- **name**: Unique identifier for the job
- **description**: Human-readable description
- **enabled**: Whether the job is active
- **schedule**: Automatic scheduling configuration
  - `type`: "systemd", "cron", or "manual"
  - `interval`: cron expression or systemd calendar
  - `enabled`: Whether scheduling is active
- **directories**: List of directories to backup
- **s3**: S3 storage configuration
- **retention**: Backup retention policy
- **skip_docker**: Skip Docker container management
- **skip_s3**: Skip S3 operations

#### Directory Configuration

- **directories**: List of directories to backup
  - `path`: Source directory path
  - `name`: Backup identifier name
  - `compression`: Enable/disable compression

- **s3**: S3 bucket configuration
  - `bucket`: S3 bucket name
  - `region`: AWS region (for AWS only, leave empty for other providers)
  - `access_key`: S3 access key
  - `secret_key`: S3 secret key
  - `endpoint`: S3-compatible endpoint URL (required for non-AWS providers)
  - `mount_point`: Local mount point
  - `use_path_style`: Use path-style S3 URLs (required for Backblaze B2)

- **retention**: Backup retention policy
  - `keep_days`: Keep backups for X days
  - `keep_count`: Keep last X backups
  - `keep_monthly`: Keep X monthly backups

## Scheduling & Automation

### Systemd Services for Multiple Jobs

```bash
# Install systemd services for all scheduled jobs
sudo backtide systemd-jobs install

# Check status of all job services
backtide systemd-jobs status

# Uninstall all job services
sudo backtide systemd-jobs uninstall
```

### Single Job Systemd Service

```bash
# Install single systemd service (legacy)
sudo backtide systemd install
```

### Cron Jobs for Multiple Jobs

```bash
# Install cron job for specific job
backtide cron install --job daily-docker-backup

# Install cron job with custom schedule
backtide cron install --job weekly-app-backup --schedule "0 3 * * 0"
```

## Usage

### Backup Operations

```bash
# Create a backup (runs default job)
backtide backup

# Run specific backup job
backtide backup daily-docker-backup

# Run all enabled backup jobs
backtide backup --all

# Create backup without Docker operations
backtide backup --skip-docker

# Run backup with specific job
backtide backup --job weekly-app-backup

# Create backup without S3 operations
backtide backup --skip-s3

# Dry run to see what would happen
backtide backup --dry-run
```

### Job Management Operations

```bash
# List all backup jobs
backtide jobs list

# Show detailed job information
backtide jobs list --detailed

# Enable a backup job
backtide jobs enable daily-docker-backup

# Disable a backup job  
backtide jobs disable weekly-app-backup

# Show job status and next run times
backtide jobs status
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

### Legacy Scheduling (Single Job)

#### Systemd Timer (Single Job)

```bash
# Install systemd service and timer
sudo backtide systemd install

# Check status
backtide systemd status

# Uninstall
sudo backtide systemd uninstall
```

#### Cron Jobs (Single Job)

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

## Interactive Job Setup

During `backtide init`, you'll be guided through creating a complete backup job:

1. **Job Configuration**: Name, description, and basic settings
2. **Schedule Selection**: Choose from daily, weekly, monthly, custom cron, or manual
3. **Retention Policy**: Configure how long to keep backups (days, count, monthly)
4. **S3 Storage**: Set up cloud storage with provider-specific defaults
5. **Docker Settings**: Choose whether to stop containers during backup
6. **Directory Selection**: Pick from common directories or add custom ones

### Schedule Options

- **Daily**: Runs every day at 2 AM
- **Weekly**: Runs every Sunday at 2 AM  
- **Monthly**: Runs on the 1st of each month at 2 AM
- **Custom Cron**: Use any cron expression
- **Manual**: No automatic scheduling

### Retention Policy Options

- **Keep Days**: Automatic deletion after X days
- **Keep Count**: Only keep X most recent backups
- **Keep Monthly**: Preserve X monthly backups for long-term storage

### S3 Configuration

During `backtide init`, you'll be guided through:

1. **Provider Selection**: Choose from AWS S3, Backblaze B2, Wasabi, DigitalOcean Spaces, MinIO, or custom
2. **Automatic Defaults**: Sensible pre-configured endpoints and settings for each provider
3. **Fstab Integration**: Option to automatically add S3 mount to `/etc/fstab` for persistence
4. **Path Style Handling**: Automatic configuration for providers requiring path-style endpoints (Backblaze B2, MinIO)

### Directory Configuration

Choose from common backup targets:
- Docker volumes (`/var/lib/docker/volumes`)
- User home directory
- System configuration (`/etc`)
- Application data (`/opt`)
- Custom directories

## Manual S3 Provider Configuration

### Supported Providers

Backtide supports any S3-compatible storage provider. Here are common configurations:

#### AWS S3 (Default)
```yaml
s3:
  bucket: my-backup-bucket
  region: us-east-1
  access_key: YOUR_ACCESS_KEY
  secret_key: YOUR_SECRET_KEY
  endpoint: ""  # Leave empty for AWS
  use_path_style: false
```

#### Backblaze B2 (Recommended)
```yaml
s3:
  bucket: my-backup-bucket
  region: ""  # Not used for B2
  access_key: YOUR_APPLICATION_KEY_ID
  secret_key: YOUR_APPLICATION_KEY
  endpoint: https://s3.us-west-002.backblazeb2.com  # Your B2 endpoint
  use_path_style: true  # REQUIRED for B2
```

#### Wasabi
```yaml
s3:
  bucket: my-backup-bucket
  region: us-east-1  # Your Wasabi region
  access_key: YOUR_ACCESS_KEY
  secret_key: YOUR_SECRET_KEY
  endpoint: https://s3.wasabisys.com  # Wasabi endpoint
  use_path_style: false
```

#### DigitalOcean Spaces
```yaml
s3:
  bucket: my-backup-bucket
  region: nyc3  # Your DO region
  access_key: YOUR_SPACES_KEY
  secret_key: YOUR_SPACES_SECRET
  endpoint: https://nyc3.digitaloceanspaces.com  # Your DO endpoint
  use_path_style: false
```

#### MinIO
```yaml
s3:
  bucket: my-backup-bucket
  region: ""  # Not used for MinIO
  access_key: YOUR_MINIO_ACCESS_KEY
  secret_key: YOUR_MINIO_SECRET_KEY
  endpoint: http://localhost:9000  # Your MinIO endpoint
  use_path_style: true  # REQUIRED for MinIO
```

## How It Works

### Job-Based Architecture

Backtide uses a job-based system where each backup job is completely independent:

```
Configuration File (~/.backtide.yaml)
├── Job: "daily-docker-backup"
│   ├── Schedule: Daily at 2 AM
│   ├── Directories: /var/lib/docker/volumes
│   ├── Retention: 7 days
│   └── S3: Backblaze B2
├── Job: "weekly-app-backup"  
│   ├── Schedule: Weekly on Sunday
│   ├── Directories: /opt/myapp/data
│   ├── Retention: 30 days
│   └── S3: AWS S3
└── Job: "monthly-archive"
    ├── Schedule: Monthly on 1st
    ├── Directories: /home/user/documents
    ├── Retention: 1 year
    └── S3: Wasabi
```

### Systemd Services Structure

When using `systemd-jobs install`:
```
/etc/systemd/system/
├── backtide-daily-docker-backup.service
├── backtide-daily-docker-backup.timer
├── backtide-weekly-app-backup.service
├── backtide-weekly-app-backup.timer
└── backtide-monthly-archive.service
    └── backtide-monthly-archive.timer

/etc/backtide/
├── daily-docker-backup.yaml
├── weekly-app-backup.yaml
└── monthly-archive.yaml
```

### Backup Process (Per Job)

1. **Container Management**: Stops all running Docker containers (optional)
2. **S3 Setup**: Installs and mounts S3 bucket using s3fs (job-specific configuration)
3. **Backup Creation**: Creates compressed tar archives of specified directories
4. **Metadata Storage**: Saves file permissions, checksums, and backup metadata
5. **Container Restoration**: Restarts previously stopped containers
6. **Cleanup**: Applies retention policies to remove old backups

### File Structure

```
/var/lib/backtide/
├── containers.json           # Docker container state
└── job_states.json          # Job execution state (future)

/mnt/backup/
├── daily-docker-backup/
│   ├── backup-2024-01-15-10-30-00/
│   │   ├── metadata.json
│   │   └── docker-volumes.tar.gz
│   └── backup-2024-01-16-10-30-00/
└── weekly-app-backup/
    ├── backup-2024-01-14-02-00-00/
    │   ├── metadata.json
    │   └── app-data.tar.gz
    └── backup-2024-01-21-02-00-00/

/etc/backtide/               # Job-specific configurations
├── daily-docker-backup.yaml
└── weekly-app-backup.yaml
```

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

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## Support

For issues and feature requests, please create an issue on the GitHub repository.