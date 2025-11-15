#!/bin/bash

# Backtide Version Management Script
# This script helps manage version numbers and create release tags

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_color() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to show usage
usage() {
    echo "Usage: $0 <command>"
    echo ""
    echo "Commands:"
    echo "  current           Show current version"
    echo "  next <type>       Calculate next version (major|minor|patch)"
    echo "  tag <version>     Create git tag for version"
    echo "  release <version> Build release binary with version"
    echo ""
    echo "Examples:"
    echo "  $0 current"
    echo "  $0 next patch"
    echo "  $0 tag 1.2.3"
    echo "  $0 release 1.2.3"
}

# Function to get current version from git tags
get_current_version() {
    local current_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "0.0.0")
    echo "${current_tag#v}"  # Remove 'v' prefix if present
}

# Function to calculate next version
calculate_next_version() {
    local current_version=$(get_current_version)
    local version_type=$1

    if [[ -z "$version_type" ]]; then
        print_color $RED "Error: Version type required (major|minor|patch)"
        usage
        exit 1
    fi

    IFS='.' read -r major minor patch <<< "$current_version"

    case $version_type in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
        *)
            print_color $RED "Error: Invalid version type '$version_type'. Use major|minor|patch"
            exit 1
            ;;
    esac

    echo "${major}.${minor}.${patch}"
}

# Function to create git tag
create_tag() {
    local version=$1

    if [[ -z "$version" ]]; then
        print_color $RED "Error: Version required"
        usage
        exit 1
    fi

    # Validate version format
    if ! [[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        print_color $RED "Error: Invalid version format '$version'. Use semantic versioning (e.g., 1.2.3)"
        exit 1
    fi

    # Check if tag already exists
    if git rev-parse "v$version" >/dev/null 2>&1; then
        print_color $RED "Error: Tag v$version already exists"
        exit 1
    fi

    print_color $BLUE "Creating tag v$version..."
    git tag -a "v$version" -m "Release v$version"
    print_color $GREEN "✓ Tag v$version created successfully"
    echo ""
    print_color $YELLOW "To push the tag, run: git push origin v$version"
}

# Function to build release binary
build_release() {
    local version=$1

    if [[ -z "$version" ]]; then
        print_color $RED "Error: Version required"
        usage
        exit 1
    fi

    print_color $BLUE "Building release binary v$version..."
    go build -ldflags="-X github.com/mitexleo/backtide/cmd.version=$version" -o backtide

    if [[ $? -eq 0 ]]; then
        print_color $GREEN "✓ Release binary built successfully"
        echo ""
        print_color $YELLOW "Binary: ./backtide"
        print_color $YELLOW "Version: $(./backtide version)"
    else
        print_color $RED "✗ Build failed"
        exit 1
    fi
}

# Main script logic
case $1 in
    current)
        current_version=$(get_current_version)
        print_color $GREEN "Current version: $current_version"
        ;;
    next)
        next_version=$(calculate_next_version "$2")
        print_color $BLUE "Next $2 version: $next_version"
        ;;
    tag)
        create_tag "$2"
        ;;
    release)
        build_release "$2"
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        if [[ -z "$1" ]]; then
            usage
        else
            print_color $RED "Error: Unknown command '$1'"
            usage
            exit 1
        fi
        ;;
esac
