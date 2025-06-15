#!/bin/sh
# shellcheck shell=dash
# shellcheck disable=SC2039  # local is non-POSIX
#
# Bash script to uninstall ast2llm-go application.

set -u # Treat unset variables as an error

# --- Global Variables ---
APP_NAME="ast2llm-go"
INSTALL_DIR_DEFAULT="$HOME/.local/bin"
RECEIPT_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/$APP_NAME"
RECEIPT_FILE="$RECEIPT_DIR/receipt.json"

INSTALLED_PATH=""

# --- Helper Functions ---

# Ensure a command exists, or exit with an error
need_cmd() {
    if ! command -v "$1" > /dev/null 2>&1; then
        err "Required command '$1' not found. Please install it."
    fi
}

say() {
    if [ "0" = "${PRINT_QUIET:-0}" ]; then
        echo "$1"
    fi
}

say_verbose() {
    if [ "1" = "${PRINT_VERBOSE:-0}" ]; then
        echo "$1"
    fi
}

warn() {
    if [ "0" = "${PRINT_QUIET:-0}" ]; then
        local yellow
        local reset
        yellow=$(tput setaf 3 2>/dev/null || echo '')
        reset=$(tput sgr0 2>/dev/null || echo '')
        say "${yellow}WARN${reset}: $1" >&2
    fi
}

err() {
    if [ "0" = "${PRINT_QUIET:-0}" ]; then
        local red
        local reset
        red=$(tput setaf 1 2>/dev/null || echo '')
        reset=$(tput sgr0 2>/dev/null || echo '')
        say "${red}ERROR${reset}: $1" >&2
    fi
    exit 1
}

ensure() {
    if ! "$@"; then
        err "Command failed: $*"
    fi
}

# --- Core Functions ---

locate_installation_info() {
    say_verbose "Locating installation information..."
    if [ -f "$RECEIPT_FILE" ]; then
        say_verbose "Found receipt file: $RECEIPT_FILE"
        # Use awk to parse JSON (assuming jq might not be present)
        INSTALLED_PATH=$(awk -F':' '/"install_path"/{gsub(/"|,/, "", $2); print $2}' "$RECEIPT_FILE" | tr -d ' ')
        if [ -z "$INSTALLED_PATH" ]; then
            warn "Could not parse install_path from receipt file. Attempting default location."
            INSTALLED_PATH="${INSTALL_DIR_DEFAULT}/${APP_NAME}"
        fi
    else
        warn "Receipt file not found: $RECEIPT_FILE. Attempting default location."
        INSTALLED_PATH="${INSTALL_DIR_DEFAULT}/${APP_NAME}"
    fi

    say "Determined installation path: $INSTALLED_PATH"

    if [ ! -f "$INSTALLED_PATH" ]; then
        err "$APP_NAME binary not found at $INSTALLED_PATH. Is it already uninstalled or installed elsewhere?"
    fi
}

remove_binary() {
    say "Removing $APP_NAME binary from $INSTALLED_PATH..."
    if [ -f "$INSTALLED_PATH" ]; then
        ensure rm "$INSTALLED_PATH"
        say "Successfully removed $APP_NAME binary."
    else
        warn "$APP_NAME binary not found at $INSTALLED_PATH. Already removed?"
    fi
}

clean_path_modifications() {
    say "Cleaning up PATH modifications..."

    local install_dir=$(dirname "$INSTALLED_PATH")
    local env_script_path="$install_dir/env"
    local env_script_path_expr="$(echo "$env_script_path" | sed "s|$HOME|\\$HOME|g")"

    # Remove source line from shell profiles
    local rc_files=".profile .bashrc .bash_profile .zshrc .zshenv"
    for rc_file in $rc_files; do
        local full_rc_path="$HOME/$rc_file"
        if [ -f "$full_rc_path" ]; then
            # Use sed in-place to remove lines that source the env script
            say_verbose "Checking $full_rc_path for source lines..."
            # Escape forward slashes in env_script_path_expr for sed
            local escaped_env_script_path_expr=$(echo "$env_script_path_expr" | sed 's/\//\\\//g')
            sed -i.bak "/\. \"${escaped_env_script_path_expr}\"/d" "$full_rc_path"
            sed -i.bak "/source \"${escaped_env_script_path_expr}\"/d" "$full_rc_path"
            rm -f "${full_rc_path}.bak" # Remove backup file
            say_verbose "Cleaned source lines in $full_rc_path."
        fi
    done

    # Remove the env script itself
    if [ -f "$env_script_path" ]; then
        say_verbose "Removing environment script: $env_script_path"
        ensure rm "$env_script_path"
    fi

    # Clean up fish shell configuration
    local fish_conf_dir="$HOME/.config/fish/conf.d"
    local fish_env_script="$fish_conf_dir/${APP_NAME}.fish"
    if [ -f "$fish_env_script" ]; then
        say_verbose "Removing Fish shell environment script: $fish_env_script"
        ensure rm "$fish_env_script"
    fi

    say "PATH modifications cleaned up."
}

remove_receipt() {
    say "Removing installation receipt..."
    if [ -f "$RECEIPT_FILE" ]; then
        ensure rm "$RECEIPT_FILE"
        say_verbose "Receipt file removed."
    else
        warn "Receipt file not found: $RECEIPT_FILE. Already removed?"
    fi

    # Remove receipt directory if it's empty
    if [ -d "$RECEIPT_DIR" ]; then
        if [ -z "$(ls -A "$RECEIPT_DIR")" ]; then # Check if directory is empty
            say_verbose "Removing empty receipt directory: $RECEIPT_DIR"
            ensure rmdir "$RECEIPT_DIR" 2>/dev/null || warn "Could not remove empty directory $RECEIPT_DIR. It might not be empty or permissions are restricted."
        fi
    fi
    say "Installation receipt cleaned."
}

# --- Main Script Execution ---

main() {
    # Parse arguments for quiet/verbose, though not strictly needed for uninstall
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --quiet)
                PRINT_QUIET=1
                ;;
            --verbose)
                PRINT_VERBOSE=1
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  --quiet             Suppress all output except errors."
                echo "  --verbose           Enable verbose output."
                exit 0
                ;;
            *)
                warn "Unknown argument: $1. Ignoring."
                ;;
        esac
        shift # Consume the argument
    done

    need_cmd rm
    need_cmd sed
    need_cmd awk
    need_cmd dirname

    locate_installation_info
    remove_binary
    clean_path_modifications
    remove_receipt

    say "$APP_NAME uninstallation complete! Please restart your shell or open a new terminal session."
}

# Run the main function
main "$@" 