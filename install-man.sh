#!/bin/bash
# install-man.sh - Install Backtide man page
#
# This script installs the Backtide man page to the system man directory.
# It can be used independently of the main installation process.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MAN_SOURCE="$SCRIPT_DIR/man/backtide.1"
MAN_TARGET="/usr/local/share/man/man1/backtide.1"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "This script requires root privileges to install man pages"
        log_info "Please run with sudo: sudo $0"
        exit 1
    fi
}

# Check if man page source exists
check_source() {
    if [ ! -f "$MAN_SOURCE" ]; then
        log_error "Man page source not found: $MAN_SOURCE"
        log_info "Make sure you're running this script from the Backtide project directory"
        exit 1
    fi
}

# Install man page
install_man_page() {
    log_info "Installing Backtide man page..."

    # Create target directory if it doesn't exist
    mkdir -p "$(dirname "$MAN_TARGET")"

    # Copy man page
    cp "$MAN_SOURCE" "$MAN_TARGET"

    # Set proper permissions
    chmod 644 "$MAN_TARGET"

    # Update man database
    log_info "Updating man database..."
    if command -v mandb >/dev/null 2>&1; then
        mandb >/dev/null 2>&1
    elif command -v makewhatis >/dev/null 2>&1; then
        makewhatis /usr/local/share/man >/dev/null 2>&1
    else
        log_warn "Could not update man database (mandb or makewhatis not found)"
        log_warn "You may need to run 'mandb' manually to update the man database"
    fi

    log_info "Man page installed successfully: $MAN_TARGET"
}

# Verify installation
verify_installation() {
    log_info "Verifying installation..."

    if [ -f "$MAN_TARGET" ]; then
        log_info "✓ Man page file exists: $MAN_TARGET"
    else
        log_error "✗ Man page file not found at: $MAN_TARGET"
        exit 1
    fi

    if man -w backtide >/dev/null 2>&1; then
        log_info "✓ Man page is accessible via 'man backtide'"
    else
        log_warn "⚠ Man page may not be accessible via 'man backtide'"
        log_warn "Try running 'mandb' to update the man database"
    fi

    log_info "Installation verification completed"
}

# Show usage information
show_usage() {
    echo "Backtide Man Page Installer"
    echo ""
    echo "Usage: $0"
    echo ""
    echo "This script installs the Backtide man page to the system."
    echo "It requires root privileges to copy files to /usr/local/share/man/"
    echo ""
    echo "Examples:"
    echo "  sudo $0          # Install man page"
    echo "  man backtide     # View installed man page"
    echo ""
    echo "Files:"
    echo "  Source: $MAN_SOURCE"
    echo "  Target: $MAN_TARGET"
}

# Main function
main() {
    local action="${1:-install}"

    case "$action" in
        install|"")
            check_root
            check_source
            install_man_page
            verify_installation
            ;;
        help|-h|--help)
            show_usage
            ;;
        *)
            log_error "Unknown action: $action"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
