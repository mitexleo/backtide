# Backtide Development Workflow Guide

## 🚀 Overview

This document outlines the development workflow for Backtide using Git Flow methodology with feature branches, automated testing, and semantic versioning.

## 📋 Branch Strategy

### Main Branches
- **`main`** - Production releases (stable, versioned)
- **`develop`** - Integration branch (pre-release, testing)

### Supporting Branches
- **`feature/`** - New features and enhancements
- **`fix/`** - Bug fixes for current release
- **`hotfix/`** - Critical production fixes

## 🔄 Development Workflow

### 1. Starting New Feature Development

```bash
# Ensure you're on develop branch
git checkout develop
git pull origin develop

# Create feature branch
git checkout -b feature/your-feature-name

# Example branch names:
# feature/add-s3-provider
# feature/improve-backup-performance
# feature/add-config-validation
```

### 2. Development Process

```bash
# Make your changes
# Write tests for new functionality
# Update documentation if needed

# Stage changes
git add .

# Commit with conventional commit message
git commit -m "feat: add support for new S3 provider"

# Or for bug fixes:
git commit -m "fix: resolve backup timing issue"

# Push feature branch
git push -u origin feature/your-feature-name
```

### 3. Testing and Quality Assurance

```bash
# Run tests locally
make test

# Build and test binary
make build
./backtide --help
./backtide version

# Check code quality
make lint

# Run specific tests
go test -v ./internal/config
go test -v ./cmd
```

### 4. Creating Pull Request

1. **Go to GitHub**: https://github.com/mitexleo/backtide
2. **Create PR**: 
   - Base: `develop` 
   - Compare: `feature/your-feature-name`
3. **PR Title**: Use conventional commit format
   - `feat: Add new S3 provider support`
   - `fix: Resolve backup timing issue`
4. **PR Description**:
   - What changes were made
   - Why they were made
   - Testing performed
   - Screenshots if applicable

### 5. Code Review Process

- **Automated Checks**: GitHub Actions will run CI tests
- **Reviewers**: At least one team member must approve
- **Feedback**: Address all review comments
- **Updates**: Push additional commits to feature branch

### 6. Merging to Develop

```bash
# Ensure CI passes and PR is approved
# Merge via GitHub UI (squash merge recommended)

# After merge, clean up local branch
git checkout develop
git pull origin develop
git branch -d feature/your-feature-name

# Delete remote branch (optional)
git push origin --delete feature/your-feature-name
```

### 7. Release Process (Automated)

When `develop` is ready for release:

```bash
# Merge develop to main
git checkout main
git pull origin main
git merge develop
git push origin main

# GitHub Actions automatically:
# - Analyzes commits for version bump
# - Creates release with semantic version
# - Builds cross-platform binaries
# - Updates CHANGELOG.md
```

## 🏗️ Build and Test Commands

### Development Builds
```bash
# Build development binary
make build

# Build for all platforms
make build-all

# Run tests
make test

# Install to system
make install

# Clean build artifacts
make clean
```

### Quality Assurance
```bash
# Format code
make fmt

# Run linters
make vet

# Format and lint
make lint

# Update dependencies
make update-deps
```

## 📝 Commit Message Convention

### Format
```
<type>: <description>

[optional body]

[optional footer]
```

### Types
- **`feat`** - New feature (triggers minor version bump)
- **`fix`** - Bug fix (triggers patch version bump)  
- **`docs`** - Documentation only changes
- **`style`** - Code style changes (formatting, etc.)
- **`refactor`** - Code refactoring
- **`test`** - Adding or updating tests
- **`chore`** - Maintenance tasks

### Examples
```bash
# Feature (minor version bump)
git commit -m "feat: add Backblaze B2 provider support"

# Fix (patch version bump)
git commit -m "fix: resolve S3 mount permission issue"

# Breaking change (major version bump)
git commit -m "feat: rewrite configuration system

BREAKING CHANGE: Configuration format changed from YAML to TOML"

# Documentation
git commit -m "docs: update installation instructions"

# Chore
git commit -m "chore: update dependencies"
```

## 🚨 Release Automation

### How Semantic Versioning Works
- **Push to `main`** → Automatic release
- **Commit Analysis**:
  - `feat:` → Minor version (1.0.0 → 1.1.0)
  - `fix:` → Patch version (1.0.0 → 1.0.1) 
  - `BREAKING CHANGE:` → Major version (1.0.0 → 2.0.0)

### Release Assets
Each release automatically includes:
- `backtide` - Linux binary
- `backtide-linux-amd64` - Linux (cross-compiled)
- `backtide-darwin-amd64` - macOS binary  
- `backtide-windows-amd64.exe` - Windows binary

## 🔧 Troubleshooting

### Common Issues

**CI Tests Failing**
```bash
# Run tests locally first
make test

# Check specific package
go test -v ./internal/backup

# Fix formatting issues
make fmt
```

**Merge Conflicts**
```bash
# Update your branch
git checkout develop
git pull origin develop
git checkout feature/your-branch
git merge develop

# Resolve conflicts, then
git add .
git commit -m "fix: resolve merge conflicts"
```

**Build Issues**
```bash
# Clean and rebuild
make clean
make build

# Check Go version
go version

# Verify dependencies
go mod verify
```

## 📊 Best Practices

### Code Quality
- Write tests for new functionality
- Run `make test` before pushing
- Use `make lint` to ensure code style
- Document public functions and types

### Git Practices
- Keep commits focused and atomic
- Use descriptive commit messages
- Rebase feature branches regularly
- Delete merged feature branches

### Release Management
- Test thoroughly on `develop` before merging to `main`
- Use conventional commits for automatic versioning
- Review CHANGELOG.md after each release
- Verify cross-platform builds work

## 🆘 Getting Help

- **Issues**: Create GitHub issues for bugs or feature requests
- **Discussions**: Use GitHub Discussions for questions
- **PR Reviews**: Request reviews from team members
- **Documentation**: Check README.md and code comments

---

*This workflow ensures consistent quality, automated releases, and collaborative development.*
```

Now let me switch back to develop branch and show you the current state: