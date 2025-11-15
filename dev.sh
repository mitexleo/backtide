#!/bin/bash

# Backtide Development Workflow Helper
# This script provides shortcuts for common development tasks

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
    echo "Backtide Development Workflow Helper"
    echo ""
    echo "Usage: ./dev.sh <command> [options]"
    echo ""
    echo "Commands:"
    echo "  start <feature>    Start new feature branch"
    echo "  test               Run tests and quality checks"
    echo "  build              Build and verify binary"
    echo "  pr                 Create PR checklist"
    echo "  release            Prepare for release"
    echo "  status             Show current development status"
    echo "  help               Show this help"
    echo ""
    echo "Examples:"
    echo "  ./dev.sh start add-s3-provider"
    echo "  ./dev.sh test"
    echo "  ./dev.sh pr"
}

# Function to start new feature
start_feature() {
    local feature_name=$1

    if [[ -z "$feature_name" ]]; then
        print_color $RED "Error: Feature name required"
        echo "Usage: ./dev.sh start <feature-name>"
        exit 1
    fi

    local branch_name="feature/$feature_name"

    print_color $BLUE "Starting new feature: $feature_name"

    # Ensure we're on develop
    git checkout develop
    git pull origin develop

    # Create feature branch
    git checkout -b "$branch_name"

    print_color $GREEN "✓ Created feature branch: $branch_name"
    print_color $YELLOW "You're now on branch: $branch_name"
    echo ""
    print_color $BLUE "Next steps:"
    echo "  1. Make your changes"
    echo "  2. Run: ./dev.sh test"
    echo "  3. Run: ./dev.sh pr (when ready)"
}

# Function to run tests
run_tests() {
    print_color $BLUE "Running quality checks and tests..."

    # Check if we're on a feature branch
    current_branch=$(git branch --show-current)
    if [[ "$current_branch" == "main" || "$current_branch" == "develop" ]]; then
        print_color $YELLOW "⚠️  You're on $current_branch branch. Consider using a feature branch."
    fi

    # Format code
    print_color $BLUE "Formatting code..."
    make fmt

    # Run linters
    print_color $BLUE "Running linters..."
    make vet

    # Run tests
    print_color $BLUE "Running tests..."
    make test

    # Build binary
    print_color $BLUE "Building binary..."
    make build

    # Test binary functionality
    print_color $BLUE "Testing binary..."
    ./backtide --help > /dev/null
    ./backtide version > /dev/null

    print_color $GREEN "✓ All checks passed!"
    echo ""
    print_color $YELLOW "Ready for commit and PR creation."
}

# Function to build and verify
build_binary() {
    print_color $BLUE "Building and verifying binary..."

    make clean
    make build

    print_color $BLUE "Testing binary functionality:"
    ./backtide version
    ./backtide --help | head -10

    print_color $GREEN "✓ Build successful!"
}

# Function to create PR checklist
create_pr_checklist() {
    current_branch=$(git branch --show-current)

    print_color $BLUE "PR Preparation Checklist for: $current_branch"
    echo ""

    # Check if we're on a feature branch
    if [[ ! "$current_branch" =~ ^feature/ ]]; then
        print_color $YELLOW "⚠️  You're not on a feature branch. PRs should typically be from feature/* branches."
    fi

    echo "✅ Before creating PR, ensure:"
    echo "   [ ] All tests pass: ./dev.sh test"
    echo "   [ ] Code is properly formatted: make fmt"
    echo "   [ ] Binary builds successfully: ./dev.sh build"
    echo "   [ ] Commit messages follow conventional format"
    echo "   [ ] Documentation updated if needed"
    echo "   [ ] No merge conflicts with develop"
    echo ""

    echo "📝 PR Creation:"
    echo "   1. Push branch: git push -u origin $current_branch"
    echo "   2. Go to: https://github.com/mitexleo/backtide/pull/new/$current_branch"
    echo "   3. Set base: develop"
    echo "   4. Use conventional commit format for title"
    echo "   5. Add detailed description"
    echo ""

    echo "🔍 PR Review Checklist:"
    echo "   [ ] Code follows project conventions"
    echo "   [ ] Tests cover new functionality"
    echo "   [ ] Documentation is updated"
    echo "   [ ] No breaking changes (unless intentional)"
    echo "   [ ] Cross-platform compatibility considered"
    echo ""

    # Check if branch is pushed
    if git ls-remote --heads origin "$current_branch" | grep -q "$current_branch"; then
        print_color $GREEN "✓ Branch is already pushed to remote"
    else
        print_color $YELLOW "⚠️  Branch not pushed yet. Run: git push -u origin $current_branch"
    fi
}

# Function to prepare for release
prepare_release() {
    print_color $BLUE "Release Preparation Checklist"
    echo ""

    # Ensure we're on develop
    current_branch=$(git branch --show-current)
    if [[ "$current_branch" != "develop" ]]; then
        print_color $RED "Error: Must be on develop branch for release preparation"
        echo "Run: git checkout develop"
        exit 1
    fi

    echo "🚀 Preparing for release from develop → main"
    echo ""

    echo "✅ Pre-release checks:"
    echo "   [ ] All PRs merged to develop"
    echo "   [ ] CI tests passing on develop"
    echo "   [ ] CHANGELOG.md updated (will be auto-updated)"
    echo "   [ ] Manual testing completed"
    echo "   [ ] Documentation reviewed"
    echo ""

    echo "📋 Release process:"
    echo "   1. Merge develop to main:"
    echo "      git checkout main"
    echo "      git pull origin main"
    echo "      git merge develop"
    echo "      git push origin main"
    echo "   2. GitHub Actions will automatically:"
    echo "      - Determine version from commits"
    echo "      - Create release with binaries"
    echo "      - Update CHANGELOG.md"
    echo ""

    # Show recent commits for version estimation
    print_color $BLUE "Recent commits (for version estimation):"
    git log --oneline -10 --decorate

    echo ""
    print_color $YELLOW "Note: Version will be automatically determined based on commit types:"
    echo "  feat: → minor version bump"
    echo "  fix:  → patch version bump"
    echo "  BREAKING CHANGE: → major version bump"
}

# Function to show development status
show_status() {
    current_branch=$(git branch --show-current)
    print_color $BLUE "Development Status - Branch: $current_branch"
    echo ""

    # Show branch info
    if [[ "$current_branch" == "main" ]]; then
        print_color $GREEN "📍 You're on main (production) branch"
    elif [[ "$current_branch" == "develop" ]]; then
        print_color $BLUE "📍 You're on develop (integration) branch"
    elif [[ "$current_branch" =~ ^feature/ ]]; then
        print_color $YELLOW "📍 You're on feature branch: $current_branch"
    else
        print_color $YELLOW "📍 You're on branch: $current_branch"
    fi

    echo ""

    # Show uncommitted changes
    if [[ -n $(git status --porcelain) ]]; then
        print_color $YELLOW "📝 You have uncommitted changes:"
        git status --porcelain
    else
        print_color $GREEN "📝 No uncommitted changes"
    fi

    echo ""

    # Show recent commits
    print_color $BLUE "Recent commits:"
    git log --oneline -5 --decorate

    echo ""

    # Show version info
    if [[ -f "./backtide" ]]; then
        print_color $BLUE "Binary version:"
        ./backtide version 2>/dev/null || echo "Not built with version info"
    fi
}

# Main script logic
case $1 in
    start)
        start_feature "$2"
        ;;
    test)
        run_tests
        ;;
    build)
        build_binary
        ;;
    pr)
        create_pr_checklist
        ;;
    release)
        prepare_release
        ;;
    status)
        show_status
        ;;
    help|--help|-h|"")
        usage
        ;;
    *)
        print_color $RED "Error: Unknown command '$1'"
        usage
        exit 1
        ;;
esac
