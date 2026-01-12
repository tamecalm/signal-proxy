#!/usr/bin/env bash
# Signal Proxy - Dependency Installer
# Installs Go dependencies for the project

set -e

# Get script directory and source libraries
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/colors.sh"
source "${SCRIPT_DIR}/lib/platform.sh"
source "${SCRIPT_DIR}/lib/utils.sh"

# Change to project root
PROJECT_ROOT="$(get_project_root)"
cd "$PROJECT_ROOT"

# Main Installation Logic
main() {
    print_banner "SIGNAL INSTALL" "Dependency Installer"
    
    # Check platform
    local os=$(detect_os)
    local arch=$(detect_arch)
    print_info "Platform: ${os}-${arch}"
    
    # Check for Go
    print_thinking "Checking for Go installation..."
    
    if ! check_go_installed; then
        print_error "Go is not installed!"
        echo ""
        echo -e "  ${HI_BLACK}Please install Go from:${RESET}"
        echo -e "  ${CYAN}https://go.dev/dl/${RESET}"
        echo ""
        exit 1
    fi
    
    local go_version=$(get_go_version)
    print_success "Go ${go_version} found"
    
    # Check for go.mod
    if [[ ! -f "go.mod" ]]; then
        print_error "go.mod not found in project root"
        print_info "Make sure you're running this from the signal-proxy directory"
        exit 1
    fi
    
    print_success "go.mod found"
    
    # Download dependencies
    print_section "Installing Dependencies"
    
    print_thinking "Downloading Go modules..."
    if go mod download 2>&1; then
        print_success "Modules downloaded"
    else
        print_error "Failed to download modules"
        exit 1
    fi
    
    print_thinking "Tidying Go modules..."
    if go mod tidy 2>&1; then
        print_success "Modules tidied"
    else
        print_warn "go mod tidy had warnings (this is usually fine)"
    fi
    
    # Verify build
    print_thinking "Verifying build..."
    if go build -o /dev/null ./cmd/proxy 2>&1; then
        print_success "Build verification passed"
    else
        print_error "Build verification failed"
        exit 1
    fi
    
    # Summary
    print_section "Installation Complete"
    
    print_group_start "Summary"
    print_group_item "Go Version" "$go_version"
    print_group_item "Platform" "${os}-${arch}"
    print_group_item "Status" "Ready to build"
    print_group_end
    
    echo -e "  ${HI_BLACK}▸${RESET} ${WHITE}Run ${CYAN}./scripts/build.sh${RESET}${WHITE} to build the project${RESET}"
    echo ""
}

# Run main function
main "$@"
