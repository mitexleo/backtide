# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project setup with Go and Cobra CLI framework
- Docker container backup functionality with permission preservation
- S3-compatible storage integration (AWS, Backblaze B2, Wasabi, DigitalOcean, MinIO)
- Job-based backup system with multiple independent configurations
- Systemd and cron scheduling support
- Retention policy management
- Interactive setup wizard (`backtide init`)
- Comprehensive backup metadata storage for disaster recovery

### Changed
- Migrated from YAML to TOML configuration format
- Separated S3 bucket management from job configurations
- Implemented bucket reusability across multiple jobs
- Enhanced validation and dependency tracking

### Technical
- Automated build and release system using GitHub Actions
- Semantic versioning with conventional commits
- Cross-platform compilation (Linux, macOS, Windows)
- Comprehensive test suite
- Version command for build information

## [1.0.0] - 2024-11-15

### Added
- Initial stable release
- Complete backup and restore functionality
- S3 bucket management commands (`backtide s3 list|add|remove|test`)
- Automated version management
- Production-ready deployment
