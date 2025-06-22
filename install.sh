#!/bin/sh
# shellcheck shell=dash
# shellcheck disable=SC2039  # local is non-POSIX
#
# Bash script to install a binary application from GitHub Releases.
# Designed for curl -LsSf <URL>/install.sh | sh installation.

set -u # Treat unset variables as an error

# --- Global Variables ---
APP_NAME="ast2llm-go"
GITHUB_REPO="ast2llm/ast2llm-go" # Owner/RepoName
INSTALL_DIR_DEFAULT="$HOME/.local/bin"
TEMP_DIR=""
NO_MODIFY_PATH=${UV_NO_MODIFY_PATH:-0}
PRINT_VERBOSE=${INSTALLER_PRINT_VERBOSE:-0}
PRINT_QUIET=${INSTALLER_PRINT_QUIET:-0}
AUTH_TOKEN="${UV_GITHUB_TOKEN:-}" # For private repositories

OS=""
ARCH=""
BINARY_SUFFIX=""
LATEST_TAG=""
COMMIT_SHA_SHORT="" # Last 4 chars of commit SHA from release
BINARY_URL=""
DOWNLOADED_BINARY_PATH=""
DOWNLOADED_CHECKSUM_PATH=""


# --- Helper Functions ---

cleanup() {
    if [ -n "${TEMP_DIR:-}" ] && [ -d "$TEMP_DIR" ]; then
        say_verbose "Cleaning up temporary directory: $TEMP_DIR"
        rm -rf "$TEMP_DIR"
    fi
}

say() {
    if [ "0" = "$PRINT_QUIET" ]; then
        echo "$1"
    fi
}

say_verbose() {
    if [ "1" = "$PRINT_VERBOSE" ]; then
        echo "$1"
    fi
}

warn() {
    if [ "0" = "$PRINT_QUIET" ]; then
        local yellow
        local reset
        yellow=$(tput setaf 3 2>/dev/null || echo '')
        reset=$(tput sgr0 2>/dev/null || echo '')
        say "${yellow}WARN${reset}: $1" >&2
    fi
}

err() {
    if [ "0" = "$PRINT_QUIET" ]; then
        local red
        local reset
        red=$(tput setaf 1 2>/dev/null || echo '')
        reset=$(tput sgr0 2>/dev/null || echo '')
        say "${red}ERROR${reset}: $1" >&2
    fi
    exit 1
}

check_cmd() {
    command -v "$1" > /dev/null 2>&1
    return $?
}

need_cmd() {
    if ! check_cmd "$1"; then
        err "Required command '$1' not found. Please install it."
    fi
}

ensure() {
    if ! "$@"; then
        err "Command failed: $*"
    fi
}

downloader() {
    local _url="$1"
    local _file="$2"
    local _dld

    if check_cmd curl; then
        _dld="curl"
    elif check_cmd wget; then
        _dld="wget"
    else
        err "Neither 'curl' nor 'wget' found. Please install one."
    fi

    say_verbose "Downloading $_url to $_file using $_dld"

    if [ "$_dld" = "curl" ]; then
        if [ -n "${AUTH_TOKEN:-}" ]; then
            ensure curl -sSfL --header "Authorization: Bearer ${AUTH_TOKEN}" "$_url" -o "$_file"
        else
            ensure curl -sSfL "$_url" -o "$_file"
        fi
    elif [ "$_dld" = "wget" ]; then
        if [ -n "${AUTH_TOKEN:-}" ]; then
            ensure wget --header "Authorization: Bearer ${AUTH_TOKEN}" "$_url" -O "$_file"
        else
            ensure wget "$_url" -O "$_file"
        fi
    fi
}

# --- Core Functions ---

detect_architecture() {
    say_verbose "Detecting OS and architecture..."
    local _ostype
    local _cputype
    _ostype="$(uname -s)"
    _cputype="$(uname -m)"

    # Handle Darwin Rosetta 2 detection
    if [ "$_ostype" = "Darwin" ]; then
        if [ "$_cputype" = "i386" ]; then
            if sysctl hw.optional.x86_64 2>/dev/null | grep -q ': 1'; then
                _cputype="x86_64"
            fi
        elif [ "$_cputype" = "x86_64" ]; then
            if sysctl hw.optional.arm64 2>/dev/null | grep -q ': 1'; then
                _cputype="arm64"
            fi
        fi
    fi

    case "$_ostype" in
        Linux) OS="linux" ;;
        Darwin) OS="darwin" ;;
        MINGW*|MSYS*|CYGWIN*|Windows_NT) OS="windows" ;;
        *) err "Unsupported OS: $_ostype" ;;
    esac

    case "$_cputype" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        i386|i686) ARCH="386" ;;
        *) err "Unsupported CPU architecture: $_cputype" ;;
    esac

    if [ "$OS" = "windows" ]; then
        BINARY_SUFFIX=".exe"
    else
        BINARY_SUFFIX=""
    fi

    say "Detected OS: $OS, Architecture: $ARCH"
}

get_latest_release_info() {
    say_verbose "Fetching latest release information from GitHub API..."
    local api_url="https://api.github.com/repos/$GITHUB_REPO/releases/latest"
    local response

    if [ -n "${AUTH_TOKEN:-}" ]; then
        response=$(curl -sSL --header "Authorization: Bearer ${AUTH_TOKEN}" "$api_url")
    else
        response=$(curl -sSL "$api_url")
    fi

    if [ -z "$response" ]; then
        err "Failed to fetch latest release information. Check repository name or network."
    fi

    LATEST_TAG=$(echo "$response" | sed -n 's/.*"tag_name": "\([^"]*\)".*/\1/p')
    if [ -z "$LATEST_TAG" ]; then
        err "Could not find latest release tag. Is the repository correct and does it have releases?"
    fi

    # Extract commit SHA from the tag_name itself (vYYYYMMDD-short_sha)
    COMMIT_SHA_SHORT=$(echo "$LATEST_TAG" | awk -F'-' '{print $NF}')
    if [ "${#COMMIT_SHA_SHORT}" -ne 4 ]; then
        warn "Could not reliably extract short commit SHA from tag: $LATEST_TAG. This might be an old or non-standard tag format."
        # If the tag format is not as expected, try to get it from the commit hash if available in the body
        # This part assumes a specific structure in the release body or asset names, which might not always hold.
        # For simplicity based on the CI's new tag format, we rely on parsing the tag.
        COMMIT_SHA_SHORT=""
    fi

    say "Latest release tag: $LATEST_TAG"
}

get_download_url() {
    say_verbose "Constructing download URL for $APP_NAME..."
    local artifact_name="${APP_NAME}-${OS}-${ARCH}${BINARY_SUFFIX}"
    download_base_url="https://github.com/$GITHUB_REPO/releases/download/$LATEST_TAG"
    BINARY_URL="${download_base_url}/${artifact_name}"
    CHECKSUM_URL="${download_base_url}/${artifact_name}.sha256"

    say "Binary URL: $BINARY_URL"
    say "Checksum URL (if available): $CHECKSUM_URL"
}

download_binary() {
    say "Downloading $APP_NAME for $OS/$ARCH..."
    TEMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'mytmpdir')
    local download_path="$TEMP_DIR/$APP_NAME"

    downloader "$BINARY_URL" "$download_path"
    DOWNLOADED_BINARY_PATH="$download_path"

    # Check if checksum file exists by attempting to head the URL
    if curl --head --silent --fail "$CHECKSUM_URL" > /dev/null; then
        say_verbose "Downloading checksum file..."
        downloader "$CHECKSUM_URL" "$DOWNLOADED_BINARY_PATH.sha256"
        DOWNLOADED_CHECKSUM_PATH="$DOWNLOADED_BINARY_PATH.sha256"
    else
        warn "SHA256 checksum file not found for ${APP_NAME}-${OS}-${ARCH}${BINARY_SUFFIX}. Skipping checksum verification."
        DOWNLOADED_CHECKSUM_PATH=""
    fi

    say "Binary downloaded to: $DOWNLOADED_BINARY_PATH"
}

verify_binary() {
    say "Verifying downloaded binary..."

    # 1. Check if it's an executable file (ELF/Mach-O/PE)
    local file_type=$(file -b "$DOWNLOADED_BINARY_PATH" 2>/dev/null || echo "")
    case "$OS" in
        linux|darwin)
            if ! echo "$file_type" | grep -qE "ELF|Mach-O"; then
                err "Downloaded file is not a valid ELF or Mach-O executable."
            fi
            ;;
        windows)
            if ! echo "$file_type" | grep -q "PE32"; then
                err "Downloaded file is not a valid PE (Windows) executable."
            fi
            ;;
    esac
    say_verbose "File type check passed: $file_type"

    # 2. Check SHA-256 checksum if available
    if [ -n "$DOWNLOADED_CHECKSUM_PATH" ] && [ -f "$DOWNLOADED_CHECKSUM_PATH" ]; then
        say_verbose "Verifying SHA-256 checksum..."
        local expected_checksum=$(awk '{print $1}' "$DOWNLOADED_CHECKSUM_PATH")
        local calculated_checksum

        if check_cmd sha256sum; then
            calculated_checksum=$(sha256sum "$DOWNLOADED_BINARY_PATH" | awk '{print $1}')
        elif check_cmd shasum; then # macOS
            calculated_checksum=$(shasum -a 256 "$DOWNLOADED_BINARY_PATH" | awk '{print $1}')
        else
            warn "Neither 'sha256sum' nor 'shasum' found. Cannot verify checksum."
            return 0
        fi

        if [ "$expected_checksum" != "$calculated_checksum" ]; then
            err "Checksum mismatch! Expected: $expected_checksum, Got: $calculated_checksum"
        else
            say "Checksum verification passed."
        fi
    else
        warn "No checksum file to verify against."
    fi
}

install_binary() {
    local target_install_dir="$1"
    say "Installing $APP_NAME to $target_install_dir..."

    ensure mkdir -p "$target_install_dir"

    # The binaries in CI are built directly, not inside archives.
    # So we just need to move and rename if necessary.
    local final_binary_name="${APP_NAME}${BINARY_SUFFIX}"
    local installed_path="${target_install_dir}/${final_binary_name}"

    say_verbose "Moving $DOWNLOADED_BINARY_PATH to $installed_path"
    ensure mv "$DOWNLOADED_BINARY_PATH" "$installed_path"
    ensure chmod +x "$installed_path"

    say "Successfully installed $APP_NAME to $installed_path"
}

generate_receipt() {
    local target_install_dir="$1"
    local installed_path="${target_install_dir}/${APP_NAME}${BINARY_SUFFIX}"

    # Generate receipt.json
    local receipt_dir="${XDG_CONFIG_HOME:-$HOME/.config}/$APP_NAME"
    ensure mkdir -p "$receipt_dir"
    local receipt_file="$receipt_dir/receipt.json"

    local installed_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local installed_version="${LATEST_TAG}"

    cat <<EOF > "$receipt_file"
{
  "app_name": "$APP_NAME",
  "version": "$installed_version",
  "install_path": "$installed_path",
  "installed_at": "$installed_at",
  "commit_hash": "$COMMIT_SHA_SHORT"
}
EOF
    say_verbose "Installation receipt written to $receipt_file"
}

install_to_path() {
    local install_path="$1"
    say "Attempting to add '$install_path' to your PATH..."

    # Check if already in PATH
    case ":$PATH:" in
        *:"$install_path":*)
            say_verbose "$install_path is already in PATH. No changes needed."
            NO_MODIFY_PATH=1
            return 0
            ;;
        *)
            ;;
    esac

    if [ "$NO_MODIFY_PATH" = "1" ]; then
        warn "Skipping PATH modification as requested or already present."
        say "To use $APP_NAME, you may need to add '$install_path' to your PATH manually."
        say "Example: export PATH=\"$install_path:$PATH\""
        return 0
    fi

    # Add to GITHUB_PATH for CI environments
    if [ -n "${GITHUB_PATH:-}" ]; then
        say_verbose "Adding '$install_path' to GITHUB_PATH for CI."
        echo "$install_path" >> "$GITHUB_PATH"
        say "PATH updated for current CI job: $install_path"
        return 0
    fi

    local env_script_path="$install_path/env"
    local env_script_path_expr="$(echo "$env_script_path" | sed "s|$HOME|\\$HOME|g")"

    # Create the env script
    say_verbose "Creating shell environment script at $env_script_path"
    cat <<EOF > "$env_script_path"
#!/bin/sh
# Add $APP_NAME binaries to PATH if they aren't added yet
case ":\${PATH}:" in
    *:"$install_path":*)
        ;;
    *)
        export PATH="$install_path:\$PATH"
        ;;
esac
EOF
    ensure chmod +x "$env_script_path"

    # Add sourcing line to shell profiles
    local rc_files=".profile .bashrc .bash_profile .zshrc .zshenv"
    local modified_any=0
    local manual_instructions=1

    for rc_file in $rc_files; do
        local full_rc_path="$HOME/$rc_file"
        if [ -f "$full_rc_path" ] || [ -d "$HOME" ]; then # Create if not exists in HOME
            # Check if source line already exists
            if ! grep -q "source \"$env_script_path_expr\"" "$full_rc_path" 2>/dev/null && ! grep -q ". \"$env_script_path_expr\"" "$full_rc_path" 2>/dev/null; then
                say_verbose "Adding source line to $full_rc_path"
                # Prepend newline in case last line of file isn't newline-terminated
                echo "" >> "$full_rc_path"
                echo ". \"$env_script_path_expr\"" >> "$full_rc_path"
                modified_any=1
            else
                say_verbose "Source line already exists in $full_rc_path."
            fi
            manual_instructions=0 # At least one file was found/modified
        fi
    done

    # Handle fish shell separately
    local fish_conf_dir="$HOME/.config/fish/conf.d"
    local fish_env_script="$fish_conf_dir/${APP_NAME}.fish"
    local fish_env_script_expr="$(echo "$fish_env_script" | sed "s|$HOME|\\$HOME|g")"

    if check_cmd fish; then
        ensure mkdir -p "$fish_conf_dir"
        say_verbose "Creating fish environment script at $fish_env_script"
        cat <<EOF > "$fish_env_script"
if not contains "$install_path" \$PATH
    set -x PATH "$install_path" \$PATH
end
EOF
        manual_instructions=0 # Fish config updated
    fi

    if [ "$modified_any" -eq 1 ]; then
        say "PATH updated in your shell configuration files. Please restart your shell or run:"
        say "    source \"$env_script_path_expr\""
    elif [ "$manual_instructions" -eq 1 ]; then
        say "To use $APP_NAME, please add '$install_path' to your PATH manually by adding the following line to your shell profile (e.g., ~/.bashrc, ~/.zshrc):"
        say "    export PATH=\"$install_path:$PATH\""
    else
        say "$APP_NAME is installed and should be available in your PATH upon next shell session."
    fi
}

# --- Main Script Execution ---

main() {
    trap cleanup EXIT # Ensure temporary files are cleaned up on exit

    local install_dir="$INSTALL_DIR_DEFAULT"

    # Parse arguments
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  --quiet             Suppress all output except errors."
                echo "  --verbose           Enable verbose output."
                echo "  --no-modify-path    Do not modify shell PATH."
                echo "  --install-dir <path>  Specify custom installation directory (default: $INSTALL_DIR_DEFAULT)."
                echo "  --self-update       Perform a self-update of the application."
                exit 0
                ;;
            --quiet)
                PRINT_QUIET=1
                ;;
            --verbose)
                PRINT_VERBOSE=1
                ;;
            --no-modify-path)
                NO_MODIFY_PATH=1
                ;;
            --install-dir)
                if [ -z "$2" ]; then
                    err "Missing argument for --install-dir"
                fi
                install_dir="$2"
                shift # Consume the value
                ;;
            --self-update)
                say "Performing self-update..."
                # Re-execute the script, effectively re-installing the latest version.
                # This assumes the script itself is downloaded via curl | sh.
                # For direct execution of an already installed script, it might be more complex.
                # For now, we assume re-running the curl | sh command.
                say "Please re-run the original installation command to get the latest version:"
                say "  curl -LsSf https://raw.githubusercontent.com/$GITHUB_REPO/main/install.sh | sh"
                exit 0
                ;;
            *)
                err "Unknown argument: $1"
                ;;
        esac
        shift # Consume the argument
    done

    # Check for necessary commands
    need_cmd curl || need_cmd wget # At least one downloader
    need_cmd file # For binary validation
    need_cmd grep
    need_cmd awk
    need_cmd mktemp
    need_cmd chmod
    need_cmd mkdir
    need_cmd rm
    # Note: 'tar' and 'unzip' are not strictly needed here as the CI pipeline
    # builds raw binaries, not archives. The old uv-installer.sh had them for archives.

    detect_architecture
    get_latest_release_info
    get_download_url
    download_binary
    verify_binary
    install_binary "$install_dir"
    generate_receipt "$install_dir" # Generate receipt after successful install
    install_to_path "$install_dir"

    say "$APP_NAME installation complete!"
}

# Run the main function
main "$@" 