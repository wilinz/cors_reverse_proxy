#!/bin/bash
# Build Windows 7 compatible binary using go-legacy-win7.
#
# Usage:
#   ./scripts/build-win7.sh                       # 32-bit console
#   GUI=1 ./scripts/build-win7.sh                 # 32-bit GUI (no console)
#   GOARCH=amd64 ./scripts/build-win7.sh          # 64-bit console
#   GUI=1 GOARCH=amd64 ./scripts/build-win7.sh    # 64-bit GUI
#
# Optional build-time defaults baked into the binary (also picked up by CI):
#   DEFAULT_TOKEN, DEFAULT_LISTENING, DEFAULT_HTTP_PROXY, DEFAULT_SKIP_TLS

set -e

APP_NAME="cors_reverse_proxy"
PKG="./cmd/cors_reverse_proxy"
CFG_PKG="cors_reverse_proxy/internal/config"
GO_LEGACY_VERSION="1.24.6-1"
INSTALL_DIR="${HOME}/go-legacy-win7-1.24"

: ${GOARCH:=386}
: ${GUI:=0}

# Detect host OS / arch for the toolchain download.
OS=$(uname -s)
ARCH=$(uname -m)
case "$OS" in
    Darwin) HOST_OS="darwin" ;;
    Linux)  HOST_OS="linux"  ;;
    *) echo "Unsupported host OS: $OS"; exit 1 ;;
esac
case "$ARCH" in
    arm64|aarch64) DOWNLOAD_ARCH="arm64" ;;
    x86_64|amd64)  DOWNLOAD_ARCH="amd64" ;;
    *) echo "Unsupported host arch: $ARCH"; exit 1 ;;
esac

echo "Host: ${HOST_OS}_${DOWNLOAD_ARCH}"

if [ ! -d "$INSTALL_DIR" ]; then
    DOWNLOAD_FILE="go-legacy-win7-${GO_LEGACY_VERSION}.${HOST_OS}_${DOWNLOAD_ARCH}.tar.gz"
    DOWNLOAD_URL="https://github.com/thongtech/go-legacy-win7/releases/download/v${GO_LEGACY_VERSION}/${DOWNLOAD_FILE}"
    echo "Downloading $DOWNLOAD_URL"
    TMP_DIR=$(mktemp -d)
    curl -L -o "$TMP_DIR/$DOWNLOAD_FILE" "$DOWNLOAD_URL"
    mkdir -p "$TMP_DIR/extract"
    tar -xzf "$TMP_DIR/$DOWNLOAD_FILE" -C "$TMP_DIR/extract"
    mkdir -p "$(dirname "$INSTALL_DIR")"
    mv "$TMP_DIR/extract/go-legacy-win7" "$INSTALL_DIR"
    rm -rf "$TMP_DIR"
    echo "Installed to $INSTALL_DIR"
else
    echo "Using existing toolchain at $INSTALL_DIR"
fi

[ -x "$INSTALL_DIR/bin/go" ] || { echo "go binary missing at $INSTALL_DIR/bin/go"; exit 1; }
"$INSTALL_DIR/bin/go" version

if [ "$GOARCH" = "386" ]; then
    ARCH_SUFFIX="x86"
else
    ARCH_SUFFIX="x64"
fi

LDFLAGS="-s -w"
if [ "$GUI" = "1" ]; then
    LDFLAGS="$LDFLAGS -H windowsgui"
    OUTPUT_FILE="${APP_NAME}_win7_${ARCH_SUFFIX}_gui.exe"
else
    OUTPUT_FILE="${APP_NAME}_win7_${ARCH_SUFFIX}.exe"
fi

# Inject build-time defaults if provided.
[ -n "$DEFAULT_TOKEN" ]      && LDFLAGS="$LDFLAGS -X '${CFG_PKG}.DefaultToken=${DEFAULT_TOKEN}'"
[ -n "$DEFAULT_LISTENING" ]  && LDFLAGS="$LDFLAGS -X '${CFG_PKG}.DefaultListening=${DEFAULT_LISTENING}'"
[ -n "$DEFAULT_HTTP_PROXY" ] && LDFLAGS="$LDFLAGS -X '${CFG_PKG}.DefaultHttpProxy=${DEFAULT_HTTP_PROXY}'"
[ -n "$DEFAULT_SKIP_TLS" ]   && LDFLAGS="$LDFLAGS -X '${CFG_PKG}.DefaultSkipTLS=${DEFAULT_SKIP_TLS}'"

echo "Building $OUTPUT_FILE (GOARCH=$GOARCH, GUI=$GUI)"
GOROOT="$INSTALL_DIR" \
GOCACHE=/tmp/go-legacy-cache \
GOMODCACHE=/tmp/go-legacy-modcache \
PATH="$INSTALL_DIR/bin:$PATH" \
GOOS=windows GOARCH=$GOARCH \
go build -ldflags="$LDFLAGS" -o "$OUTPUT_FILE" "$PKG"

echo "✓ $OUTPUT_FILE ($(du -h "$OUTPUT_FILE" | cut -f1))"
