# Backtide - Docker Backup Utility

A comprehensive backup utility for Docker-based applications with S3 integration, built in Go with Cobra CLI framework.

## 🚨 Critical Features for Production Use

- **Disaster Recovery Ready**: All metadata stored in S3 - server loss doesn't mean backup loss
- **Permission Preservation**: Exact file permissions and ownership restored (PostgreSQL 999:999, etc.)
- **Single Systemd Service**: Efficient single service for all backup jobs
- **Job-Based Architecture**: Multiple independent backup configurations
- **Cross-Platform S3 Support**: AWS, Backblaze B2, Wasabi, DigitalOcean, MinIO, and more

## Features

- **Multi-directory Backup**: Backup multiple directories with compression
- **Docker Container Management**: Stop and restart containers during backup operations
- **S3FS Integration**: Automatic S3FS installation and S3 bucket mounting
- **Permission Preservation**: **Exact file permissions and ownership restored** (PostgreSQL 999:999, MySQL, etc.)
- **Retention Policies**: Configurable backup retention (days, count, monthly)
- **Scheduling Options**: Both systemd timer and cron job support
- **Metadata Tracking**: **Comprehensive backup metadata stored in S3 for disaster recovery**
- **No Database Required**: File-based state management
- **Job-Based System**: Multiple independent backup configurations
- **Cross-Platform S3**: AWS, Backblaze B2, Wasabi, DigitalOcean, MinIO, and more
- **S3-Only Mode**: No local storage by default (prevents disk exhaustion)
- **Smart Systemd**: Single service reads config file directly, no regeneration needed

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
nano ~/.backtide.toml  # Edit manually
```

## Job-Based Backup System

Backtide now supports multiple backup jobs with different configurations:

- **Multiple Jobs**: Run different backups for different purposes
- **Independent Scheduling**: Each job can have its own schedule
- **Flexible Configuration**: Different S3 buckets, directories, and retention per job
- **Granular Control**: Enable/disable jobs individually
- **Unique Job IDs**: Each job has a unique identifier for tracking
- **Single Systemd Service**: Efficient single service runs all scheduled jobs
- **No Service Regeneration**: Systemd service reads config file directly
- **Easy Restart**: Simple restart command for configuration changes
- **S3-Only Default**: No local storage to prevent disk exhaustion

### Example Multi-Job Configuration:
```toml
[[jobs]]
id = 'job-20240115-143000'
name = 'daily-docker-backup'
description = 'Daily backup of Docker volumes'
enabled = true
bucket_id = ''
skip_docker = false
skip_s3 = false

[jobs.schedule]
type = 'systemd'
interval = 'daily'
enabled = true

[[jobs.directories]]
path = '/var/lib/docker/volumes'
name = 'docker-volumes'
compression = true

[jobs.retention]
keep_days = 7
keep_count = 7
keep_monthly = 0

[jobs.storage]
local = false
s3 = true

[[jobs]]
id = 'job-20240115-143001'
name = 'weekly-app-backup'
description = 'Weekly backup of application data'
enabled = true
bucket_id = ''
skip_docker = true
skip_s3 = false

[jobs.schedule]
type = 'systemd'
interval = 'weekly'
enabled = true

[[jobs.directories]]
path = '/opt/myapp/data'
name = 'app-data'
compression = true

[jobs.retention]
keep_days = 30
keep_count = 4
keep_monthly = 6

[jobs.storage]
local = false
s3 = true
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

```toml
# ~/.backtide.toml
backup_path = ''  # Empty = S3-only mode (recommended)
temp_path = '/tmp/backtide'

[[jobs]]
id = 'job-default'
name = 'default-backup'
description = 'Default backup job'
enabled = true
bucket_id = ''
skip_docker = false
skip_s3 = false

[jobs.schedule]
type = 'systemd'
interval = 'daily'
enabled = true

[[jobs.directories]]
path = '/var/lib/docker/volumes'
name = 'docker-volumes'
compression = true

[[jobs.directories]]
path = '/opt/myapp/data'
name = 'app-data'
compression = true

[jobs.retention]
keep_days = 30
keep_count = 10
keep_monthly = 6

[jobs.storage]
local = false
s3 = true

# S3 bucket configuration (separate from jobs)
[[buckets]]
id = 'bucket-default'
name = 'my-backup-bucket'
bucket = 'my-backup-bucket'
region = 'us-east-1'
access_key = 'YOUR_ACCESS_KEY'
secret_key = 'YOUR_SECRET_KEY'
endpoint = ''
mount_point = '/mnt/s3backup'
use_path_style = false
provider = 'AWS S3'
description = 'Default backup bucket'
```

### TOML Configuration Format

Backtide now uses TOML format for configuration files (`~/.backtide.toml`). TOML provides:

- **Better readability**: Clear, predictable syntax
- **No whitespace sensitivity**: Unlike YAML, indentation doesn't matter
- **Type safety**: Explicit typing prevents configuration errors
- **Array of tables**: Clean syntax for lists of objects using `[[ ]]`

#### Understanding Double Brackets `[[ ]]`

Double brackets `[[ ]]` in TOML create **arrays of tables** (lists of objects):

```toml
# Array of job objects
[[jobs]]
id = 'job-1'
name = 'first-job'

[[jobs]]
id = 'job-2'
name = 'second-job'

# Array of directory objects within each job
[[jobs.directories]]
path = '/path1'
name = 'dir1'

[[jobs.directories]]
path = '/path2'
name = 'dir2'

# Array of bucket objects
[[buckets]]
id = 'bucket-1'
name = 'first-bucket'

[[buckets]]
id = 'bucket-2'
name = 'second-bucket'
```

This is equivalent to YAML's list syntax but more explicit and less error-prone.

### Job Configuration Options

Each backup job supports:

- **id**: Unique identifier for the job (auto-generated)
- **name**: Human-readable identifier for the job
- **description**: Human-readable description
- **enabled**: Whether the job is active
- **schedule**: Automatic scheduling configuration
  - `type`: "systemd", "cron", or "manual"
  - `interval`: cron expression or systemd calendar
  - `enabled`: Whether scheduling is active
- **directories**: List of directories to backup
- **bucket_id**: Reference to S3 bucket configuration
- **retention**: Backup retention policy
- **skip_docker**: Skip Docker container management
- **skip_s3**: Skip S3 operations
- **storage**: Storage configuration (local, S3, or both)

#### Directory Configuration

- **directories**: List of directories to backup
  - `path`: Source directory path
  - `name`: Backup identifier name
  - `compression`: Enable/disable compression

#### S3 Bucket Configuration (Separate Management)

S3 buckets are now managed separately from jobs and can be reused:

- **buckets**: Array of S3 bucket configurations
  - `id`: Unique bucket identifier
  - `name`: Human-readable bucket name
  - `bucket`: S3 bucket name
  - `region`: AWS region (for AWS only, leave empty for other providers)
  - `access_key`: S3 access key
  - `secret_key`: S3 secret key
  - `endpoint`: S3-compatible endpoint URL (required for non-AWS providers)
  - `mount_point`: Local mount point
  - `use_path_style`: Use path-style S3 URLs (required for Backblaze B2)
  - `provider`: Provider type (AWS S3, Backblaze B2, etc.)
  - `description`: Optional description

- **retention**: Backup retention policy
  - `keep_days`: Keep backups for X days
  - `keep_count`: Keep last X backups
  - `keep_monthly`: Keep X monthly backups

#### Storage Configuration

- **storage**: Storage mode configuration
  - `local`: Enable local storage (requires `backup_path`)
  - `s3`: Enable S3 storage (requires `bucket_id` reference)

## Scheduling & Automation

### Systemd Service for All Jobs (Smart Design)

```bash
# Install single systemd service for all backup jobs
sudo backtide systemd-jobs install

# Restart service after configuration changes
sudo backtide systemd-jobs restart

# Check status of the service
backtide systemd-jobs status

# Uninstall the service
sudo backtide systemd-jobs uninstall

# Check service status
backtide systemd-jobs status
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

## S3 Bucket Management Features

Backtide now supports separate S3 bucket management with the following benefits:

- **Bucket Reusability**: Single bucket configuration can serve multiple backup jobs
- **Separate Configuration**: Buckets managed independently from job settings
- **Dependency Tracking**: Prevents removal of buckets used by active jobs
- **Provider Support**: All major S3-compatible providers (AWS, Backblaze B2, Wasabi, etc.)
- **Mount Point Management**: Each bucket has independent mount configuration

### Key Features:

1. **Bucket List with Usage Counts**: See which jobs use each bucket
2. **Interactive Bucket Creation**: Guided setup for all major providers
3. **Safe Removal**: Dependency checking prevents orphaned job references
4. **Connectivity Testing**: Test bucket access and permissions
5. **Provider Defaults**: Automatic configuration for common providers

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

### S3 Bucket Management

```bash
# List all S3 bucket configurations
backtide s3 list

# Add a new S3 bucket configuration
backtide s3 add

# Remove a bucket configuration
backtide s3 remove bucket-name

# Force remove without confirmation
backtide s3 remove bucket-name --force

# Test bucket connectivity
backtide s3 test bucket-name
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

### Storage Modes

#### S3-Only Mode (Recommended)
- **No local storage**: Backups go directly to S3
- **Prevents disk exhaustion**: No risk of filling up server storage
- **Disaster recovery ready**: All metadata stored in S3
- **Default configuration**: `backup_path: ""`

#### Local + S3 Mode
- **Local copies**: Backups stored locally and in S3
- **Faster restores**: Local copies for quick recovery
- **Requires disk space**: Set `backup_path: "/mnt/backup"`

### Systemd Service Management

#### Smart Systemd Design
- **Single service**: One service runs all backup jobs
- **No regeneration**: Service reads config file directly
- **Easy updates**: Modify config, then restart service
- **Efficient**: No complex multi-service management

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
```toml
[[buckets]]
id = 'bucket-aws'
name = 'AWS Backup Bucket'
bucket = 'my-backup-bucket'
region = 'us-east-1'
access_key = 'YOUR_ACCESS_KEY'
secret_key = 'YOUR_SECRET_KEY'
endpoint = ''  # Leave empty for AWS
mount_point = '/mnt/s3backup'
use_path_style = false
provider = 'AWS S3'
description = 'AWS S3 backup bucket'
```

#### Backblaze B2 (Recommended)
```toml
[[buckets]]
id = 'bucket-b2'
name = 'Backblaze B2 Bucket'
bucket = 'my-backup-bucket'
region = ''  # Not used for B2
access_key = 'YOUR_APPLICATION_KEY_ID'
secret_key = 'YOUR_APPLICATION_KEY'
endpoint = 'https://s3.us-west-002.backblazeb2.com'  # Your B2 endpoint
mount_point = '/mnt/s3backup'
use_path_style = true  # REQUIRED for B2
provider = 'Backblaze B2'
description = 'Backblaze B2 backup bucket'
```

#### Wasabi
```toml
[[buckets]]
id = 'bucket-wasabi'
name = 'Wasabi Bucket'
bucket = 'my-backup-bucket'
region = 'us-east-1'  # Your Wasabi region
access_key = 'YOUR_ACCESS_KEY'
secret_key = 'YOUR_SECRET_KEY'
endpoint = 'https://s3.wasabisys.com'  # Wasabi endpoint
mount_point = '/mnt/s3backup'
use_path_style = false
provider = 'Wasabi'
description = 'Wasabi backup bucket'
```

#### DigitalOcean Spaces
```toml
[[buckets]]
id = 'bucket-do'
name = 'DigitalOcean Spaces'
bucket = 'my-backup-bucket'
region = 'nyc3'  # Your DO region
access_key = 'YOUR_SPACES_KEY'
secret_key = 'YOUR_SPACES_SECRET'
endpoint = 'https://nyc3.digitaloceanspaces.com'  # Your DO endpoint
mount_point = '/mnt/s3backup'
use_path_style = false
provider = 'DigitalOcean Spaces'
description = 'DigitalOcean Spaces bucket'
```

#### MinIO
```toml
[[buckets]]
id = 'bucket-minio'
name = 'MinIO Bucket'
bucket = 'my-backup-bucket'
region = ''  # Not used for MinIO
access_key = 'YOUR_MINIO_ACCESS_KEY'
secret_key = 'YOUR_MINIO_SECRET_KEY'
endpoint = 'http://localhost:9000'  # Your MinIO endpoint
mount_point = '/mnt/s3backup'
use_path_style = true  # REQUIRED for MinIO
provider = 'MinIO'
description = 'MinIO backup bucket'
```

## How It Works

### Job-Based Architecture

Backtide uses a job-based system where each backup job is completely independent:

```
Configuration File (~/.backtide.toml)
├── Buckets (Separate Management)
│   ├── "aws-backup-bucket" (AWS S3)
│   ├── "b2-backup-bucket" (Backblaze B2)
│   └── "wasabi-bucket" (Wasabi)
└── Jobs (Reference Buckets by ID)
    ├── Job: "daily-docker-backup"
    │   ├── Schedule: Daily at 2 AM
    │   ├── Directories: /var/lib/docker/volumes
    │   ├── Retention: 7 days
    │   └── Bucket: "b2-backup-bucket"
    ├── Job: "weekly-app-backup"  
    │   ├── Schedule: Weekly on Sunday
    │   ├── Directories: /opt/myapp/data
    │   ├── Retention: 30 days
    │   └── Bucket: "aws-backup-bucket"
    └── Job: "monthly-archive"
        ├── Schedule: Monthly on 1st
        ├── Directories: /home/user/documents
        ├── Retention: 1 year
        └── Bucket: "wasabi-bucket"
```

### Systemd Service Structure

When using `systemd-jobs install`:
```
/etc/systemd/system/
├── backtide.service          # Single service for all jobs
└── backtide.timer            # Single timer for scheduled execution

~/.backtide.toml              # Central configuration for all jobs and buckets
/etc/systemd/system/
├── backtide.service          # Single service reads config directly
└── backtide.timer            # Timer for scheduled execution
```

**Smart Design**: Single service reads configuration file directly - no need to regenerate systemd files when jobs change. Just modify the config and restart the service.

### Storage Architecture

#### S3-Only Mode (Default)
```
Backup Process:
1. Create backup in temp directory
2. Copy backup files to S3
3. Store metadata in S3 for disaster recovery
4. Clean up temp directory

Benefits:
- No local storage consumption
- Server loss doesn't affect backups
- All restore information in cloud
```

#### Local + S3 Mode
```
Backup Process:
1. Create backup in local directory
2. Copy metadata to S3 for disaster recovery
3. Optional: Copy backup files to S3

Benefits:
- Faster local restores
- Redundant storage
- Requires local disk space
```

### Backup Process (Per Job)

1. **Container Management**: Stops all running Docker containers (optional)
2. **S3 Setup**: Installs and mounts S3 bucket using s3fs (job-specific configuration)
3. **Permission Scanning**: Extracts exact file permissions, UID/GID for all files
4. **Backup Creation**: Creates compressed tar archives of specified directories
5. **Metadata Storage**: **Saves file permissions, checksums, and backup metadata to S3**
6. **Container Restoration**: Restarts previously stopped containers
7. **Cleanup**: Applies retention policies to remove old backups

### File Structure (S3-Only Mode)

```
/var/lib/backtide/
├── containers.json           # Docker container state (temporary)

/tmp/backtide/                # Temporary backup creation
└── backup-2024-01-15-10-30-00/
    ├── metadata.json         # Temporary metadata
    └── docker-volumes.tar.gz # Temporary backup file

/mnt/s3backup/                # S3 mount point (primary storage)
├── backtide-metadata/        # Disaster recovery storage
│   ├── backup-2024-01-15-10-30-00/
│   │   └── metadata.json     # S3 metadata (critical for recovery)
│   └── backup-2024-01-14-02-00-00/
│       └── metadata.json     # S3 metadata
└── backups/                  # Backup file storage
    ├── backup-2024-01-15-10-30-00/
    │   └── docker-volumes.tar.gz
    └── backup-2024-01-14-02-00-00/
        └── app-data.tar.gz

/mnt/s3backup/                # S3 mount point
└── backtide-metadata/        # Disaster recovery storage
    ├── backup-2024-01-15-10-30-00/
    │   └── metadata.json     # S3 metadata (critical for recovery)
    └── backup-2024-01-14-02-00-00/
        └── metadata.json     # S3 metadata

~/.backtide.toml              # Central configuration for all jobs
```

**Disaster Recovery**: Critical metadata stored in S3 ensures backups survive server loss.

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

## Build and Release

Backtide uses automated GitHub Actions workflows for building and releasing.

### Automated Builds

- **CI Workflow**: Runs on every push to `main` and pull requests
- **Release Workflow**: Creates releases when version tags are pushed
- **Manual Releases**: Can be triggered from GitHub UI with custom version

### Building Locally

```bash
# Build development version
make build

# Build release version with specific version
make release VERSION=1.2.3

# Cross-compile for all platforms
make build-all

# Run tests
make test

# Install to system
make install
```

### Version Management

```bash
# Show current version
./scripts/version.sh current

# Calculate next version
./scripts/version.sh next patch

# Create git tag for release
./scripts/version.sh tag 1.2.3

# Build release binary
./scripts/version.sh release 1.2.3
```

### Release Process

1. **Create Release Tag**:
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```

2. **GitHub Actions** automatically:
   - Builds the binary with the correct version
   - Runs tests
   - Creates a GitHub release
   - Uploads the binary

3. **Manual Release** (optional):
   - Go to GitHub Actions → Release workflow
   - Click "Run workflow"
   - Enter version number (e.g., 1.2.3)

### Version Command

The built binary includes version information:
```bash
backtide version
# Output: Backtide version 1.2.3
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
backtide --config /path/to/config.toml backup
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## Support

For issues and feature requests, please create an issue on the GitHub repository.