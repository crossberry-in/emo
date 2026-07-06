#!/usr/bin/env bash
#
# emo installer — downloads and installs the emo CLI from GitHub releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/crossberry-in/emo/main/install.sh | bash
#
# Or to install a specific version:
#   curl -fsSL https://raw.githubusercontent.com/crossberry-in/emo/main/install.sh | bash -s -- v0.1.0
#
set -euo pipefail

REPO="crossberry-in/emo"
VERSION="${1:-latest}"

# Color output (if terminal)
if [ -t 1 ]; then
    BOLD='\033[1m'
    GREEN='\033[0;32m'
    RED='\033[0;31m'
    YELLOW='\033[0;33m'
    NC='\033[0m'
else
    BOLD=''; GREEN=''; RED=''; YELLOW=''; NC=''
fi

info()  { echo -e "${GREEN}✓${NC} $*"; }
warn()  { echo -e "${YELLOW}!${NC} $*"; }
error() { echo -e "${RED}✗${NC} $*" >&2; }

# ---------------------------------------------------------------------------
# Fallback: build from source using Go.
# Called when pre-built binary download fails.
# Args: $1 = binary name
# ---------------------------------------------------------------------------
install_from_source() {
    local BINARY_NAME="${1:-emo}"

    if ! command -v go >/dev/null 2>&1; then
        error "Go is not installed. Please install Go 1.22+ from https://go.dev/dl/"
        error "  Or download a pre-built binary from:"
        error "  https://github.com/$REPO/releases"
        exit 1
    fi

    if ! command -v git >/dev/null 2>&1; then
        error "git is not installed. Please install git first."
        exit 1
    fi

    TMP_SRC=$(mktemp -d)

    echo "Cloning $REPO…"
    if ! git clone --depth 1 "https://github.com/$REPO.git" "$TMP_SRC/emo" 2>&1; then
        error "git clone failed."
        rm -rf "$TMP_SRC"
        exit 1
    fi

    echo "Building emo (this requires Go 1.22+)…"
    INSTALL_DIR="${EMO_INSTALL_DIR:-$HOME/.local/bin}"
    mkdir -p "$INSTALL_DIR"

    (cd "$TMP_SRC/emo" && go build -ldflags="-s -w" -o "$INSTALL_DIR/$BINARY_NAME" ./cmd/emo) || {
        error "go build failed."
        rm -rf "$TMP_SRC"
        exit 1
    }

    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    rm -rf "$TMP_SRC"
    info "Built and installed to $INSTALL_DIR/$BINARY_NAME"

    case ":$PATH:" in
        *":$INSTALL_DIR:"*) ;;
        *)
            warn "$INSTALL_DIR is not in your PATH."
            echo "  Add: export PATH=\"\$HOME/.local/bin:\$PATH\" to ~/.bashrc"
            ;;
    esac

    if "$INSTALL_DIR/$BINARY_NAME" --help >/dev/null 2>&1; then
        info "emo installed successfully!"
        echo ""
        echo -e "  ${BOLD}Quick start:${NC}"
        echo "    emo init myapp"
        echo "    cd myapp && emo start"
        echo ""
        echo -e "  ${BOLD}Docs:${NC} https://github.com/$REPO"
        exit 0
    else
        error "Build succeeded but 'emo --help' failed."
        exit 1
    fi
}

# --- Detect OS ---
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux)  OS="linux";;
    Darwin) OS="darwin";;
    MINGW*|MSYS*|CYGWIN*) OS="windows";;
    *) error "Unsupported OS: $OS"; exit 1;;
esac

case "$ARCH" in
    x86_64|amd64) ARCH="amd64";;
    arm64|aarch64) ARCH="arm64";;
    *) error "Unsupported architecture: $ARCH"; exit 1;;
esac

echo -e "${BOLD}emo installer${NC}"
echo "  OS:       $OS"
echo "  Arch:     $ARCH"
echo "  Version:  $VERSION"
echo ""

# --- Resolve version ---
# NOTE: We temporarily disable 'set -e' for version resolution because
# curl returns non-zero on HTTP 403 (rate limit), which would kill the
# script under 'set -e'. We handle failures by checking if VERSION is empty.
set +e
if [ "$VERSION" = "latest" ]; then
    echo "Resolving latest version…"
    API_URL="https://api.github.com/repos/$REPO/releases/latest"
    if [ -n "${GITHUB_TOKEN:-}" ]; then
        VERSION=$(curl -fsSL -H "Authorization: token $GITHUB_TOKEN" "$API_URL" 2>/dev/null | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    fi
    if [ -z "$VERSION" ] || [ "$VERSION" = "latest" ]; then
        VERSION=$(curl -fsSL "$API_URL" 2>/dev/null | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    fi
    if [ -z "$VERSION" ] || [ "$VERSION" = "latest" ]; then
        VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/git/refs/tags" 2>/dev/null | grep '"ref"' | tail -1 | sed -E 's/.*refs\/tags\/([^"]+)".*/\1/')
    fi
    if [ -z "$VERSION" ] || [ "$VERSION" = "latest" ]; then
        VERSION="v0.1.2"
        warn "Could not reach GitHub API (rate limit?). Using $VERSION."
    else
        info "Latest version: $VERSION"
    fi
fi
set -e

# --- Download binary ---
BINARY_NAME="emo"
if [ "$OS" = "windows" ]; then
    BINARY_NAME="emo.exe"
fi

ASSET_NAME="emo-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    ASSET_NAME="emo-${OS}-${ARCH}.exe"
fi

# Try GitHub release asset URL first.
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$ASSET_NAME"

echo "Downloading $DOWNLOAD_URL…"
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

HTTP_CODE=$(curl -fsSL -w "%{http_code}" -o "$TMP_DIR/$BINARY_NAME" "$DOWNLOAD_URL" || true)
if [ "$HTTP_CODE" != "200" ]; then
    warn "Pre-built binary not found for $OS/$ARCH ($HTTP_CODE)."
    warn "Building from source…"
    install_from_source "$BINARY_NAME"
    exit 0
fi

info "Downloaded binary"

# --- Install ---
INSTALL_DIR="${EMO_INSTALL_DIR:-$HOME/.local/bin}"
mkdir -p "$INSTALL_DIR"

# Don't overwrite an existing installation without warning.
if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    warn "Existing installation found at $INSTALL_DIR/$BINARY_NAME — overwriting."
fi

mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

info "Installed to $INSTALL_DIR/$BINARY_NAME"

# --- PATH check ---
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
        echo ""
        warn "$INSTALL_DIR is not in your PATH."
        echo "  Add this line to your ~/.bashrc or ~/.zshrc:"
        echo ""
        echo -e "    ${BOLD}export PATH=\"\$HOME/.local/bin:\$PATH\"${NC}"
        echo ""
        echo "  Then run: source ~/.bashrc  (or restart your terminal)"
        ;;
esac

# --- Verify ---
echo ""
if "$INSTALL_DIR/$BINARY_NAME" --help >/dev/null 2>&1; then
    info "emo installed successfully!"
    echo ""
    echo -e "  ${BOLD}Quick start:${NC}"
    echo "    emo init myapp"
    echo "    cd myapp"
    echo "    emo start"
    echo ""
    echo -e "  ${BOLD}With a template:${NC}"
    echo "    emo templates"
    echo "    emo init myapp --template counter"
    echo ""
    echo -e "  ${BOLD}Install a component:${NC}"
    echo "    emo components"
    echo "    emo install Card"
    echo ""
    echo -e "  ${BOLD}Docs:${NC} https://github.com/$REPO"
    exit 0
else
    error "Installation completed but 'emo --help' failed. Please report this issue."
    exit 1
fi
