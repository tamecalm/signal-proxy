#!/usr/bin/env bash
# Signal Proxy - Platform Detection Library
# Auto-detects OS and architecture for cross-platform builds

# OS Detection
# Detect the current operating system
# Returns: linux, darwin, windows
detect_os() {
    local os=""
    
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        CYGWIN*)    os="windows" ;;
        MINGW*)     os="windows" ;;
        MSYS*)      os="windows" ;;
        *)          os="unknown" ;;
    esac
    
    echo "$os"
}

# Get human-readable OS name
get_os_display_name() {
    local os="$1"
    
    case "$os" in
        linux)      echo "Linux" ;;
        darwin)     echo "macOS" ;;
        windows)    echo "Windows" ;;
        *)          echo "Unknown" ;;
    esac
}

# Architecture Detection
# Detect the current CPU architecture
# Returns: amd64, arm64, arm, 386
detect_arch() {
    local arch=""
    
    case "$(uname -m)" in
        x86_64)     arch="amd64" ;;
        amd64)      arch="amd64" ;;
        arm64)      arch="arm64" ;;
        aarch64)    arch="arm64" ;;
        armv7l)     arch="arm" ;;
        armv6l)     arch="arm" ;;
        i686)       arch="386" ;;
        i386)       arch="386" ;;
        *)          arch="unknown" ;;
    esac
    
    echo "$arch"
}

# Get human-readable architecture name
get_arch_display_name() {
    local arch="$1"
    
    case "$arch" in
        amd64)      echo "64-bit (x86_64)" ;;
        arm64)      echo "ARM 64-bit" ;;
        arm)        echo "ARM 32-bit" ;;
        386)        echo "32-bit (x86)" ;;
        *)          echo "Unknown" ;;
    esac
}

# Binary Naming
# Get the binary file extension for the OS
get_binary_extension() {
    local os="$1"
    
    if [[ "$os" == "windows" ]]; then
        echo ".exe"
    else
        echo ""
    fi
}

# Get the full binary name with platform suffix
# Usage: get_binary_name [os] [arch]
# If no args provided, uses current platform
get_binary_name() {
    local os="${1:-$(detect_os)}"
    local arch="${2:-$(detect_arch)}"
    local ext=$(get_binary_extension "$os")
    
    echo "signal-proxy-${os}-${arch}${ext}"
}

# Get the build output path
# Usage: get_build_output_path [os] [arch]
get_build_output_path() {
    local os="${1:-$(detect_os)}"
    local arch="${2:-$(detect_arch)}"
    local binary_name=$(get_binary_name "$os" "$arch")
    
    echo "build/${binary_name}"
}

# Platform Validation
# Check if the detected platform is supported
is_platform_supported() {
    local os="${1:-$(detect_os)}"
    local arch="${2:-$(detect_arch)}"
    
    # Supported combinations
    case "${os}-${arch}" in
        linux-amd64)    return 0 ;;
        linux-arm64)    return 0 ;;
        linux-arm)      return 0 ;;
        darwin-amd64)   return 0 ;;
        darwin-arm64)   return 0 ;;
        windows-amd64)  return 0 ;;
        windows-386)    return 0 ;;
        *)              return 1 ;;
    esac
}

# Get list of all supported platforms
get_supported_platforms() {
    echo "linux-amd64"
    echo "linux-arm64"
    echo "darwin-amd64"
    echo "darwin-arm64"
    echo "windows-amd64"
}

# Environment Detection
# Detect if running in a CI environment
is_ci() {
    if [[ -n "${CI:-}" ]] || [[ -n "${GITHUB_ACTIONS:-}" ]] || [[ -n "${GITLAB_CI:-}" ]] || [[ -n "${JENKINS_URL:-}" ]]; then
        return 0
    fi
    return 1
}

# Detect if running in a container
is_container() {
    if [[ -f /.dockerenv ]] || grep -q 'docker\|lxc' /proc/1/cgroup 2>/dev/null; then
        return 0
    fi
    return 1
}

# Get platform info as a formatted string
get_platform_info() {
    local os=$(detect_os)
    local arch=$(detect_arch)
    local os_display=$(get_os_display_name "$os")
    local arch_display=$(get_arch_display_name "$arch")
    
    echo "${os_display} ${arch_display} (${os}-${arch})"
}

# Print platform summary (for display)
print_platform_summary() {
    local os=$(detect_os)
    local arch=$(detect_arch)
    
    # Source colors if not already loaded
    if [[ -z "${COLORS_ENABLED:-}" ]]; then
        local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
        source "${script_dir}/colors.sh"
    fi
    
    print_group_start "Platform Detection"
    print_group_item "Operating System" "$(get_os_display_name "$os")"
    print_group_item "Architecture" "$(get_arch_display_name "$arch")"
    print_group_item "Target" "${os}-${arch}"
    print_group_item "Binary Name" "$(get_binary_name)"
    print_group_end
}
