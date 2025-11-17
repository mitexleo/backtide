# Backtide - Docker Backup Utility

[![GitHub Release](https://img.shields.io/github/v/release/mitexleo/backtide)](https://github.com/mitexleo/backtide/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/mitexleo/backtide)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A powerful backup utility designed specifically for Docker-based applications, featuring S3FS integration, metadata preservation, and automated scheduling.

## 📋 Table of Contents

- [🚀 Quick Start](#-quick-start)
- [✨ Features](#-features)
- [📦 Installation](#-installation)
- [⚙️ Configuration](#️-configuration)
- [🔧 Usage](#-usage)
- [🏗️ Architecture](#️-architecture)
- [🔒 Security](#-security)
- [🛠️ Development](#️-development)
- [❓ Troubleshooting](#-troubleshooting)
- [🤝 Contributing](#-contributing)

## 🚀 Quick Start

### 1. Installation
```bash
# Download latest release
wget https://github.com/mitexleo/backtide/releases/latest/download/backtide-linux-amd64
sudo mv backtide-linux-amd64 /usr/local/bin/backtide
sudo chmod +x /usr/local/bin/backtide
```

### 2. Initialize Configuration
```bash
sudo backtide init
```

### 3. Add S3 Bucket
```bash
sudo backtide s3 add
```

### 4. Create Backup Job
```bash
backtide jobs add
```

### 5. Run First Backup
```bash
backtide backup
```

## ✨ Features

### 🎯 Core Features
- **Multi-job backup system** - Configure multiple independent backup jobs
- **Docker container management** - Automatic stop/start during backup
- **S3FS integration** - Direct S3 bucket mounting for cloud storage
- **Metadata preservation** - File permissions, ownership, and timestamps
- **Compression support** - Gzip compression for efficient storage
- **Retention policies** - Automatic cleanup of old backups
- **Cross-platform** - Linux, macOS, and Windows support

### 🔄 Automation
- **Systemd services** - Native Linux service management
- **Cron integration** - Traditional scheduling support
- **Smart scheduling** - Multiple job coordination
- **Self-updating** - Automatic binary updates

### ☁️ Cloud Storage
- **Multiple S3 providers** - AWS, Backblaze B2, Wasabi, DigitalOcean, MinIO
- **Path/Virtual host style** - Configurable endpoint styles
- **Credential isolation** - Separate credentials per bucket
- **Persistent mounts** - Automatic fstab configuration

## 📦 Installation

### Prerequisites
- **Go 1.19+** (for building from source)
- **s3fs-fuse** (for S3 bucket mounting)
- **Docker** (for container backup functionality)

### System Packages
```bash
# Ubuntu/Debian
sudo apt-get install s3fs

# CentOS/RHEL
sudo yum install s3fs-fuse

# Fedora
sudo dnf install s3fs-fuse

# Alpine
sudo apk add s3fs-fuse
```

### Build from Source
```bash
git clone https://github.com/mitexleo/backtide
cd backtide
make build
sudo make install
```

### Binary Installation
```bash
# Download latest release
curl -s https://api.github.com/repos/mitexleo/backtide/releases/latest | \
  grep "browser_download_url.*linux-amd64" | \
  cut -d '"' -f 4 | \
  wget -qi - -O backtide
sudo mv backtide /usr/local/bin/
sudo chmod +x /usr/local/bin/backtide
```

## ⚙️ Configuration

### System Configuration Location
Backtide uses a centralized system configuration in `/etc/backtide/`:

```
/etc/backtide/
├── config.toml              # Main configuration
└── s3-credentials/          # Secure credential storage
    ├── passwd-s3fs-bucket-1
    └── passwd-s3fs-bucket-2
```

### Configuration Structure
```toml
# /etc/backtide/config.toml
backup_path = "/var/lib/backtide"
temp_path = "/tmp/backtide"

[[buckets]]
id = "bucket-production"
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
id = "job-docker-backup"
name = "Docker Volumes Backup"
description = "Backup all Docker volumes"
enabled = true
bucket_id = "bucket-production"

[jobs.schedule]
type = "daily"
interval = "24h"
enabled = true

[[jobs.directories]]
path = "/var/lib/docker/volumes"
name = "docker-volumes"
compression = true

[jobs.retention]
keep_days = 30
keep_count = 10
keep_monthly = 6

[jobs.storage]
local = false
s3 = true
```

### S3 Provider Configuration

#### AWS S3
```toml
[[buckets]]
id = "aws-bucket"
name = "AWS S3 Production"
bucket = "my-backup-bucket"
region = "us-east-1"
access_key = "AKIAIOSFODNN7EXAMPLE"
secret_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
endpoint = ""
mount_point = "/mnt/s3backup-aws"
use_path_style = false
provider = "AWS S3"
```

#### Backblaze B2
```toml
[[buckets]]
id = "b2-bucket"
name = "Backblaze B2 Backup"
bucket = "my-b2-bucket"
region = ""
access_key = "0021a1b2c3d4e5f6789012345abcdefg0000000001"
secret_key = "K0021a1b2c3d4e5f6789012345abcdefg0000000001"
endpoint = "https://s3.us-west-002.backblazeb2.com"
mount_point = "/mnt/s3backup-b2"
use_path_style = true  # Recommended for B2
provider = "Backblaze B2"
```

#### MinIO
```toml
[[buckets]]
id = "minio-bucket"
name = "MinIO Development"
bucket = "my-minio-bucket"
region = ""
access_key = "minioadmin"
secret_key = "minioadmin"
endpoint = "http://localhost:9000"
mount_point = "/mnt/s3backup-minio"
use_path_style = true  # Required for MinIO
provider = "MinIO"
```

## 🔧 Usage

### Backup Operations
```bash
# Run all enabled backup jobs
backtide backup

# Run specific job
backtide backup --job "Docker Volumes Backup"

# Dry run (show what would be backed up)
backtide backup --dry-run

# Force backup (ignore schedule)
backtide backup --force
```

### Job Management
```bash
# List all jobs
backtide jobs list

# Add new job interactively
backtide jobs add

# Show job details
backtide jobs show "Docker Volumes Backup"

# Enable/disable job
backtide jobs enable "Docker Volumes Backup"
backtide jobs disable "Docker Volumes Backup"
```

### S3 Bucket Management
```bash
# List configured buckets
backtide s3 list

# Add new bucket interactively
sudo backtide s3 add

# Test bucket connectivity
backtide s3 test bucket-id

# Remove bucket configuration
sudo backtide s3 remove bucket-id
```

### Restore Operations
```bash
# List available backups
backtide list

# Restore specific backup
backtide restore backup-2024-01-15-10-30-00

# Restore to different location
backtide restore backup-2024-01-15-10-30-00 --target /restore/location
```

### System Management
```bash
# Clean up old backups
backtide cleanup

# Update to latest version
backtide update

# Show version information
backtide version

# Initialize system configuration
sudo backtide init
```

## 🏗️ Architecture

### System Design
Backtide uses a modular architecture with clear separation of concerns:

- **Configuration Layer** - TOML-based configuration management
- **S3FS Integration** - Cloud storage mounting and management
- **Backup Engine** - Core backup/restore functionality
- **Scheduler** - Job scheduling and automation
- **CLI Interface** - User interaction and command processing

### File Structure
```
/var/lib/backtide/          # Backup storage (local mode)
├── job-docker-backup/
│   ├── backup-2024-01-15-10-30-00/
│   │   ├── metadata.toml
│   │   ├── docker-volumes.tar.gz
│   │   └── app-data.tar.gz
│   └── backup-2024-01-14-10-30-00/
└── job-app-backup/
    └── backup-2024-01-15-10-30-00/

/mnt/s3backup/              # S3 mount point (S3 mode)
└── backtide/
    ├── job-docker-backup/
    └── job-app-backup/
```

### Backup Process
1. **Pre-backup checks** - Verify configuration and dependencies
2. **Docker container management** - Stop containers if configured
3. **Directory backup** - Compress and backup configured directories
4. **Metadata preservation** - Save file permissions and ownership
5. **S3 upload** - Transfer to cloud storage (S3 mode)
6. **Cleanup** - Remove temporary files, restart containers

## 🔒 Security

### Credential Management
- **Isolated credentials** - Each bucket has separate credential files
- **Secure storage** - Credentials stored in `/etc/backtide/s3-credentials/` with `0600` permissions
- **No credential sharing** - Buckets cannot access each other's credentials
- **Automatic cleanup** - Credentials removed when buckets are deleted

### File Permissions
```bash
/etc/backtide/config.toml           # 0644 - Readable by all
/etc/backtide/s3-credentials/       # 0700 - Owner only
/etc/backtide/s3-credentials/*      # 0600 - Owner read/write only
```

### Best Practices
1. **Use IAM roles** when possible instead of access keys
2. **Regular credential rotation** for S3 providers
3. **Monitor backup logs** for unauthorized access attempts
4. **Secure configuration backups** of `/etc/backtide/`

## 🛠️ Development

### Project Structure
```
backtide/
├── cmd/                 # CLI command implementations
│   ├── root.go         # Main command entry point
│   ├── backup.go       # Backup operations
│   ├── s3.go           # S3 bucket management
│   └── jobs.go         # Job management
├── internal/           # Internal packages
│   ├── config/         # Configuration management
│   ├── s3fs/           # S3FS integration
│   └── backup/         # Core backup engine
├── main.go             # Application entry point
└── Makefile           # Build and development tasks
```

### Building from Source
```bash
# Clone repository
git clone https://github.com/mitexleo/backtide
cd backtide

# Build binary
make build

# Run tests
make test

# Install system-wide
sudo make install

# Clean build artifacts
make clean
```

### Development Workflow
```bash
# Create feature branch
git checkout -b feature/your-feature-name

# Make changes and test
make test
./backtide --help

# Commit with conventional format
git commit -m "feat: add new S3 provider support"

# Create pull request to develop branch
```

### Testing
```bash
# Run all tests
go test ./...

# Test specific package
go test -v ./internal/config

# Run with coverage
go test -cover ./...

# Integration tests
go test -tags=integration ./...
```

## ❓ Troubleshooting

### Common Issues

**S3 Mount Failures**
```bash
# Check s3fs installation
which s3fs

# Test bucket connectivity
backtide s3 test bucket-id

# Check credentials
sudo cat /etc/backtide/s3-credentials/passwd-s3fs-bucket-id

# Verify fstab entry
cat /etc/fstab | grep s3fs
```

**Permission Errors**
```bash
# Ensure proper sudo usage
sudo backtide s3 add
sudo backtide init

# Check directory permissions
ls -la /etc/backtide/
ls -la /etc/backtide/s3-credentials/
```

**Backup Failures**
```bash
# Enable verbose logging
backtide backup --verbose

# Check job configuration
backtide jobs list

# Verify S3 connectivity
backtide s3 list
```

**Update Issues**
```bash
# Manual update if automatic fails
sudo backtide update --force

# Check current version
backtide version

# Download manually from releases page
```

### Debug Mode
```bash
# Enable debug output
backtide --verbose backup

# Check system logs
journalctl -u backtide
journalctl -u backtide@job-name

# View backup logs
tail -f /var/log/backtide/backtide.log
```

## 🤝 Contributing

We welcome contributions! Please see our development guidelines:

1. **Fork the repository**
2. **Create a feature branch** (`git checkout -b feature/amazing-feature`)
3. **Commit your changes** using conventional commit format
4. **Push to the branch** (`git push origin feature/amazing-feature`)
5. **Open a Pull Request**

### Commit Convention
```
<type>: <description>

[optional body]

[optional footer]
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

### Development Setup
```bash
# Set up development environment
git clone https://github.com/mitexleo/backtide
cd backtide

# Install dependencies
go mod download

# Run tests
make test

# Build development binary
make build
```

### Reporting Issues
When reporting issues, please include:
- Backtide version (`backtide version`)
- Operating system and version
- Configuration details (redacted)
- Error messages and logs
- Steps to reproduce

---

**Backtide** - Reliable backups for Docker applications. Built with ❤️ using Go.

For support, create an issue on [GitHub](https://github.com/mitexleo/backtide/issues).