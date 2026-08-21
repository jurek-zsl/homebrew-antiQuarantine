#!/usr/bin/env bash
# ==============================================================================
# antiQuarantine (aq) Standalone One-Liner Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/jurek-zsl/homebrew-antiQuarantine/main/install.sh | bash
# ==============================================================================

set -euo pipefail

# ANSI color codes
BOLD="$(tput bold 2>/dev/null || printf '')"
GREEN="$(tput setaf 2 2>/dev/null || printf '')"
BLUE="$(tput setaf 4 2>/dev/null || printf '')"
YELLOW="$(tput setaf 3 2>/dev/null || printf '')"
RED="$(tput setaf 1 2>/dev/null || printf '')"
RESET="$(tput sgr0 2>/dev/null || printf '')"

REPO="jurek-zsl/homebrew-antiQuarantine"
TAG="v2.0.0"
VERSION="2.0.0"

echo "${BLUE}${BOLD}🛡️  antiQuarantine (aq) Installer${RESET}"
echo "----------------------------------------------------"

# 1. Platform Check (macOS only)
OS="$(uname -s)"
if [ "$OS" != "Darwin" ]; then
    echo "${RED}Error: antiQuarantine is only supported on macOS (Darwin). Detected OS: $OS${RESET}" >&2
    exit 1
fi

# 2. Architecture Detection
ARCH="$(uname -m)"
case "$ARCH" in
    arm64|aarch64)
        ARCH_NAME="arm64"
        ;;
    x86_64|amd64)
        ARCH_NAME="amd64"
        ;;
    *)
        echo "${YELLOW}Warning: Unknown architecture '$ARCH', defaulting to universal release.${RESET}"
        ARCH_NAME="all"
        ;;
esac

echo "Detected Platform: macOS ($ARCH_NAME)"

# 3. Determine Installation Destination
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

if [ ! -d "$INSTALL_DIR" ]; then
    if [ "$INSTALL_DIR" = "/usr/local/bin" ]; then
        echo "Creating $INSTALL_DIR (may require sudo)..."
        sudo mkdir -p "$INSTALL_DIR"
    else
        mkdir -p "$INSTALL_DIR"
    fi
fi

# 4. Download & Extract
TMP_DIR="$(mktemp -d /tmp/aq_install_XXXXXX)"
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/aq_${VERSION}_darwin_${ARCH_NAME}.tar.gz"
# Fallback to universal binary tarball if arch-specific not found
if ! curl --head --silent --fail "$DOWNLOAD_URL" > /dev/null; then
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/aq_${VERSION}_darwin_all.tar.gz"
fi

echo "Downloading ${BOLD}aq ${TAG}${RESET} from GitHub Releases..."
curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/aq.tar.gz"

echo "Extracting binary..."
tar -xzf "$TMP_DIR/aq.tar.gz" -C "$TMP_DIR"

if [ ! -f "$TMP_DIR/aq" ]; then
    echo "${RED}Error: Failed to find 'aq' binary in release archive.${RESET}" >&2
    exit 1
fi

# 5. Install Binary
echo "Installing to ${BOLD}${INSTALL_DIR}/aq${RESET}..."
if [ -w "$INSTALL_DIR" ]; then
    install -m 755 "$TMP_DIR/aq" "$INSTALL_DIR/aq"
else
    echo "${YELLOW}Writing to $INSTALL_DIR requires administrator privileges:${RESET}"
    sudo install -m 755 "$TMP_DIR/aq" "$INSTALL_DIR/aq"
fi

# 6. Verify Installation
if command -v aq >/dev/null 2>&1; then
    INSTALLED_VER="$(aq --version 2>/dev/null || echo '2.0.0')"
    echo ""
    echo "${GREEN}${BOLD}✨ Successfully installed antiQuarantine ($INSTALLED_VER)!${RESET}"
    echo ""
    echo "Quick Start:"
    echo "  ${BLUE}aq check /path/to/app.app${RESET}      # Check quarantine status"
    echo "  ${BLUE}aq inspect ~/Downloads/file.zip${RESET} # Deep inspect metadata & origin URL"
    echo "  ${BLUE}aq strip ~/Downloads/file.zip${RESET}   # Strip quarantine attribute"
    echo "  ${BLUE}aq tui ~/Downloads${RESET}              # Interactive terminal browser"
    echo ""
else
    echo ""
    echo "${GREEN}${BOLD}✨ Binary installed to ${INSTALL_DIR}/aq!${RESET}"
    echo "${YELLOW}Note: Ensure ${INSTALL_DIR} is in your PATH:${RESET}"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    echo ""
fi
