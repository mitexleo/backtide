#!/bin/bash
# generate-man.sh - Generate man page from markdown documentation
#
# This script converts the README.md documentation into a properly formatted
# man page using pandoc or generates a basic man page if pandoc is not available.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR"
MAN_DIR="$PROJECT_ROOT/man"
README="$PROJECT_ROOT/README.md"
MAN_PAGE="$MAN_DIR/backtide.1"

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

# Check if pandoc is available
check_pandoc() {
    if command -v pandoc >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Generate man page using pandoc
generate_with_pandoc() {
    log_info "Generating man page using pandoc..."

    pandoc -s -f markdown -t man "$README" -o "$MAN_PAGE" \
        --metadata title="BACKTIDE" \
        --metadata section="1" \
        --metadata date="$(date +'%B %Y')" \
        --metadata footer="Backtide 1.0" \
        --variable header="Backtide User Manual" \
        --variable adjust-margin=5

    if [ $? -eq 0 ]; then
        log_info "Man page generated successfully with pandoc: $MAN_PAGE"
        return 0
    else
        log_error "Failed to generate man page with pandoc"
        return 1
    fi
}

# Generate basic man page from template
generate_basic_man() {
    log_warn "pandoc not found, generating basic man page from template..."

    cat > "$MAN_PAGE" << 'EOF'
.TH BACKTIDE 1 "January 2025" "Backtide 1.0" "Backup Utility Manual"
.SH NAME
backtide \- comprehensive backup utility for Docker applications with S3 integration
.SH SYNOPSIS
.B backtide
[\fIOPTIONS\fR] \fICOMMAND\fR [\fIARGS\fR]...
.br
.B backtide
[\fIOPTIONS\fR] backup [\fI--job\fR \fIJOB-NAME\fR | \fI--all\fR]
.br
.B backtide
[\fIOPTIONS\fR] restore \fIBACKUP-ID\fR [\fI--target\fR \fIPATH\fR]
.br
.B backtide
[\fIOPTIONS\fR] list [\fI--backups\fR | \fI--jobs\fR]
.br
.B backtide
[\fIOPTIONS\fR] delete \fIBACKUP-ID\fR [\fI--force\fR]
.br
.B backtide
[\fIOPTIONS\fR] jobs \fISUBCOMMAND\fR [\fIARGS\fR]...
.br
.B backtide
[\fIOPTIONS\fR] s3 \fISUBCOMMAND\fR [\fIARGS\fR]...
.br
.B backtide
[\fIOPTIONS\fR] init
.br
.B backtide
[\fIOPTIONS\fR] update
.br
.B backtide
[\fIOPTIONS\fR] version
.SH DESCRIPTION
.B backtide
is a powerful backup utility designed specifically for Docker-based applications,
featuring S3FS integration, metadata preservation, and automated scheduling.
.PP
For complete documentation, see the README.md file or visit:
.B https://github.com/mitexleo/backtide
.SH OPTIONS
.TP
.BR \-c ", " \-\-config " " \fIFILE\fR
Specify configuration file (default: /etc/backtide/config.toml or ~/.backtide.toml)
.TP
.BR \-v ", " \-\-verbose
Enable verbose output for detailed logging
.TP
.BR \-\-dry\-run
Show what would be done without making changes
.TP
.BR \-f ", " \-\-force
Force operation, skip confirmation prompts
.TP
.BR \-h ", " \-\-help
Show help information
.SH COMMANDS
.SS "Backup Operations"
.TP
.BR backup
Run backup operations
.TP
.BR restore
Restore specific backup
.TP
.BR list
List available backups or configured jobs
.TP
.BR delete
Delete specific backup or manage backup cleanup
.TP
.BR cleanup
Clean up old backups according to retention policies
.SS "Job Management"
.TP
.BR jobs
Manage backup job configurations
.SS "S3 Bucket Management"
.TP
.BR s3
Manage S3 bucket configurations (requires root)
.SS "System Management"
.TP
.BR init
Initialize system configuration (requires root)
.TP
.BR update
Update to latest version
.TP
.BR version
Show version information
.TP
.BR systemd
Manage systemd services for automated backups
.TP
.BR cron
Manage cron jobs for automated backups
.TP
.BR daemon
Start scheduling daemon for automated backups
.SH EXAMPLES
.TP
.B Basic backup operations
.nf
# Run all enabled backup jobs
backtide backup

# Run specific job
backtide backup --job "Docker Volumes Backup"

# Dry run to see what would be backed up
backtide backup --dry-run
.fi
.TP
.B Restore operations
.nf
# List available backups
backtide list --backups

# Restore specific backup (as root for ownership)
sudo backtide restore backup-2024-01-15-10-30-00
.fi
.TP
.B System setup
.nf
# Initialize system configuration
sudo backtide init

# Update to latest version
backtide update
.fi
.SH FILES
.TP
.B /etc/backtide/config.toml
System-wide configuration file
.TP
.B /etc/backtide/s3-credentials/
Directory containing S3 credential files
.TP
.B ~/.backtide.toml
User-specific configuration file
.TP
.B /var/lib/backtide/
Default backup storage location
.SH NOTES
.IP \(bu 3
Backtide is designed for Linux systems only
.IP \(bu 3
Root privileges are required for ownership preservation during restore
.IP \(bu 3
Docker must be installed and running for container management
.IP \(bu 3
s3fs-fuse must be installed for S3 bucket mounting
.SH BUGS
Report bugs at the GitHub repository: https://github.com/mitexleo/backtide/issues
.SH SEE ALSO
.BR docker (1),
.BR s3fs (1),
.BR systemctl (1),
.BR cron (8)
.SH AUTHORS
Backtide was developed by mitexleo and contributors.
EOF

    log_info "Basic man page generated: $MAN_PAGE"
}

# Main function
main() {
    log_info "Starting man page generation..."

    # Create man directory if it doesn't exist
    mkdir -p "$MAN_DIR"

    # Check if README exists
    if [ ! -f "$README" ]; then
        log_error "README.md not found at: $README"
        exit 1
    fi

    # Try to generate with pandoc first, fall back to basic generation
    if check_pandoc; then
        generate_with_pandoc
    else
        generate_basic_man
    fi

    # Validate the generated man page
    if [ -f "$MAN_PAGE" ]; then
        log_info "Man page validation:"
        echo "  File: $MAN_PAGE"
        echo "  Size: $(wc -l < "$MAN_PAGE") lines"
        echo "  First few lines:"
        head -5 "$MAN_PAGE"

        log_info "Man page generation completed successfully!"
        log_info "You can view it with: man ./man/backtide.1"
    else
        log_error "Man page generation failed - no output file created"
        exit 1
    fi
}

# Run main function
main "$@"
