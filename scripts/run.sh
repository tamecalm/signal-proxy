#!/usr/bin/env bash
# Signal Proxy - Run Script
# Runs the built proxy binary for the current platform

set -e

# Get script directory and source libraries
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/colors.sh"
source "${SCRIPT_DIR}/lib/platform.sh"
source "${SCRIPT_DIR}/lib/utils.sh"

# Change to project root
PROJECT_ROOT="$(get_project_root)"
cd "$PROJECT_ROOT"

# Build directory
BUILD_DIR="${PROJECT_ROOT}/build"

# Help Text
show_help() {
    echo ""
    echo "Usage: run.sh [OPTIONS] [-- PROXY_ARGS]"
    echo ""
    echo "Options:"
    echo "  --dev           Run with development environment"
    echo "  --prod          Run with production environment"
    echo "  --build-first   Build before running"
    echo "  --help          Show this help message"
    echo ""
    echo "Examples:"
    echo "  ./run.sh                      # Run the proxy"
    echo "  ./run.sh --dev                # Run in development mode"
    echo "  ./run.sh --build-first        # Build then run"
    echo "  ./run.sh -- --config custom.json"
    echo ""
}

# Main Run Logic
main() {
    local build_first=false
    local env_mode=""
    local proxy_args=()
    local parsing_proxy_args=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        if [[ "$parsing_proxy_args" == true ]]; then
            proxy_args+=("$1")
            shift
            continue
        fi
        
        case "$1" in
            --dev)
                env_mode="development"
                shift
                ;;
            --prod)
                env_mode="production"
                shift
                ;;
            --build-first)
                build_first=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            --)
                parsing_proxy_args=true
                shift
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    print_banner "SIGNAL RUN" "Proxy Launcher"
    
    # Detect platform
    local os=$(detect_os)
    local arch=$(detect_arch)
    local binary_name=$(get_binary_name "$os" "$arch")
    local binary_path="${BUILD_DIR}/${binary_name}"
    
    print_info "Platform: ${os}-${arch}"
    
    # Build if requested or binary doesn't exist
    if [[ "$build_first" == true ]] || [[ ! -f "$binary_path" ]]; then
        if [[ ! -f "$binary_path" ]]; then
            print_warn "Binary not found, building first..."
        else
            print_info "Building before run..."
        fi
        
        "${SCRIPT_DIR}/build.sh"
        echo ""
    fi
    
    # Verify binary exists and is executable
    if [[ ! -f "$binary_path" ]]; then
        print_error "Binary not found: ${binary_path}"
        print_info "Run ./scripts/build.sh to build the project"
        exit 1
    fi
    
    # Make sure it's executable
    chmod +x "$binary_path" 2>/dev/null || true
    
    print_success "Binary found: ${binary_name}"
    
    # Set environment if specified
    if [[ -n "$env_mode" ]]; then
        export APP_ENV="$env_mode"
        print_info "Environment: ${env_mode}"
    fi
    
    # Run the proxy
    print_section "Starting Signal Proxy"
    print_separator
    echo ""
    
    # Execute the binary
    exec "$binary_path" "${proxy_args[@]}"
}

# Run main function
main "$@"
