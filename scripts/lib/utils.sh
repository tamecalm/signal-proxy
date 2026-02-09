#!/usr/bin/env bash
# Signal Proxy - Utility Functions Library
# Common helper functions for build scripts

# Get the project root directory
get_project_root() {
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    echo "$(cd "${script_dir}/../.." && pwd)"
}

# Dependency Checks
# Check if Go is installed
check_go_installed() {
    if command -v go &> /dev/null; then
        return 0
    fi
    return 1
}

# Get Go version
get_go_version() {
    if check_go_installed; then
        go version | awk '{print $3}' | sed 's/go//'
    else
        echo "not installed"
    fi
}

# Check if a command exists
check_command() {
    local cmd="$1"
    if command -v "$cmd" &> /dev/null; then
        return 0
    fi
    return 1
}

# Check all required dependencies
check_dependencies() {
    local missing=()
    
    if ! check_go_installed; then
        missing+=("go")
    fi
    
    if ! check_command "git"; then
        missing+=("git")
    fi
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        echo "${missing[@]}"
        return 1
    fi
    
    return 0
}

# Directory Management
# Ensure a directory exists
ensure_dir() {
    local dir="$1"
    if [[ ! -d "$dir" ]]; then
        mkdir -p "$dir"
    fi
}

# Clean a directory (remove and recreate)
clean_dir() {
    local dir="$1"
    if [[ -d "$dir" ]]; then
        rm -rf "$dir"
    fi
    mkdir -p "$dir"
}

# User Interaction
# Prompt for yes/no confirmation
# Usage: confirm_prompt "Are you sure?" [default: y/n]
confirm_prompt() {
    local message="$1"
    local default="${2:-n}"
    local prompt_suffix
    
    if [[ "$default" == "y" ]]; then
        prompt_suffix="[Y/n]"
    else
        prompt_suffix="[y/N]"
    fi
    
    read -r -p "$message $prompt_suffix " response
    response=${response:-$default}
    
    case "$response" in
        [yY][eE][sS]|[yY]) return 0 ;;
        *) return 1 ;;
    esac
}

# Display a menu and get user selection
# Usage: show_menu "Title" "Option 1" "Option 2" ...
show_menu() {
    local title="$1"
    shift
    local options=("$@")
    
    # Source colors if not already loaded
    if [[ -z "${COLORS_ENABLED:-}" ]]; then
        local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
        source "${script_dir}/colors.sh"
    fi
    
    print_section "$title"
    echo ""
    
    local i=1
    for option in "${options[@]}"; do
        echo -e "  ${CYAN}${i})${RESET} ${WHITE}${option}${RESET}"
        ((i++))
    done
    
    echo ""
    read -r -p "  Select option [1-${#options[@]}]: " choice
    
    if [[ "$choice" =~ ^[0-9]+$ ]] && [[ "$choice" -ge 1 ]] && [[ "$choice" -le "${#options[@]}" ]]; then
        echo "$choice"
        return 0
    else
        echo "0"
        return 1
    fi
}

# File Operations
# Check if a file exists
file_exists() {
    [[ -f "$1" ]]
}

# Check if file is executable
is_executable() {
    [[ -x "$1" ]]
}

# Get file size in human readable format
get_file_size() {
    local file="$1"
    if [[ -f "$file" ]]; then
        local size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
        format_bytes "$size"
    else
        echo "N/A"
    fi
}

# Format bytes to human readable
format_bytes() {
    local bytes="$1"
    
    if [[ -z "$bytes" ]] || [[ "$bytes" == "0" ]]; then
        echo "0B"
    elif [[ $bytes -lt 1024 ]]; then
        echo "${bytes}B"
    elif [[ $bytes -lt 1048576 ]]; then
        echo "$(awk "BEGIN {printf \"%.1f\", $bytes/1024}")KB"
    elif [[ $bytes -lt 1073741824 ]]; then
        echo "$(awk "BEGIN {printf \"%.1f\", $bytes/1048576}")MB"
    else
        echo "$(awk "BEGIN {printf \"%.1f\", $bytes/1073741824}")GB"
    fi
}

# Error Handling
# Exit with error message
die() {
    local message="$1"
    local code="${2:-1}"
    
    # Source colors if not already loaded
    if [[ -z "${COLORS_ENABLED:-}" ]]; then
        local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
        source "${script_dir}/colors.sh"
    fi
    
    print_error "$message"
    exit "$code"
}

# Trap handler for cleanup on exit
setup_cleanup_trap() {
    local cleanup_fn="$1"
    trap "$cleanup_fn" EXIT INT TERM
}

# Timing
# Get current timestamp in seconds
get_timestamp() {
    date +%s
}

# Calculate elapsed time and format nicely
format_elapsed() {
    local start="$1"
    local end="${2:-$(get_timestamp)}"
    local elapsed=$((end - start))
    
    if [[ $elapsed -lt 60 ]]; then
        echo "${elapsed}s"
    else
        local mins=$((elapsed / 60))
        local secs=$((elapsed % 60))
        echo "${mins}m ${secs}s"
    fi
}
