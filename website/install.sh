#!/bin/sh
set -eu

REPO="khanhtd36/lazygit"
BIN="lazygit"
INSTALL_DIR="${LAZYGIT_INSTALL_DIR:-$HOME/.local/bin}"
API_URL="https://api.github.com/repos/${REPO}/releases/latest"

main() {
    OS="$(uname -s)"
    case "$OS" in
        Linux)  os="linux" ;;
        Darwin) os="darwin" ;;
        *)      err "unsupported OS: $OS" ;;
    esac

    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)   arch="x86_64" ;;
        aarch64|arm64)  arch="arm64" ;;
        *)              err "unsupported architecture: $ARCH" ;;
    esac

    log "detected ${os}/${arch}"

    need curl
    need tar

    log "fetching latest release info..."
    TAG="$(curl -fsSL --retry 3 --connect-timeout 10 --max-time 20 "$API_URL" \
        | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/')" \
        || err "can't reach ${API_URL}. Please try again later."
    if [ -z "$TAG" ]; then
        err "couldn't determine latest release tag from ${API_URL}"
    fi
    VERSION="${TAG#v}"

    ASSET="lazygit_${VERSION}_${os}_${arch}.tar.gz"
    BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"

    log "downloading ${TAG} (${ASSET})..."
    TMP="$(mktemp -d)"
    trap 'rm -rf "$TMP"' EXIT

    curl -fsSL --retry 3 --connect-timeout 10 --max-time 120 "${BASE_URL}/${ASSET}" -o "${TMP}/${ASSET}" \
        || err "download failed from ${BASE_URL}/${ASSET}"
    curl -fsSL --retry 3 --connect-timeout 10 --max-time 20 "${BASE_URL}/checksums.txt" -o "${TMP}/checksums.txt" \
        || err "download failed from ${BASE_URL}/checksums.txt"

    EXPECTED_SHA256="$(awk -v f="$ASSET" '$2 == f { print $1 }' "${TMP}/checksums.txt")"
    if [ -z "$EXPECTED_SHA256" ]; then
        err "checksums.txt does not include an entry for ${ASSET}"
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL_SHA256="$(sha256sum "${TMP}/${ASSET}" | awk '{ print $1 }')"
    elif command -v shasum >/dev/null 2>&1; then
        ACTUAL_SHA256="$(shasum -a 256 "${TMP}/${ASSET}" | awk '{ print $1 }')"
    else
        err "SHA-256 verification requires sha256sum or shasum"
    fi
    if [ "$ACTUAL_SHA256" != "$EXPECTED_SHA256" ]; then
        err "downloaded lazygit checksum did not match"
    fi

    tar -xzf "${TMP}/${ASSET}" -C "$TMP" "$BIN"

    mkdir -p "$INSTALL_DIR"
    mv "${TMP}/${BIN}" "${INSTALL_DIR}/${BIN}"
    chmod +x "${INSTALL_DIR}/${BIN}"

    log "installed ${BIN} ${VERSION} to ${INSTALL_DIR}/${BIN}"

    case ":${PATH}:" in
        *":${INSTALL_DIR}:"*) ;;
        *)
            echo ""
            warn "${INSTALL_DIR} is not in your PATH"
            echo "  add it to your shell config:"
            echo ""
            echo "    export PATH=\"${INSTALL_DIR}:\$PATH\""
            echo ""
            ;;
    esac

    if command -v "$BIN" >/dev/null 2>&1; then
        log "ready. run 'lazygit' to get started."
    fi
}

log()  { printf '  \033[32m>\033[0m %s\n' "$1"; }
warn() { printf '  \033[33m!\033[0m %s\n' "$1"; }
err()  { printf '  \033[31m\xe2\x9c\x97\033[0m %s\n' "$1" >&2; exit 1; }

need() {
    if ! command -v "$1" >/dev/null 2>&1; then
        err "requires '$1' — install it first, or download a binary manually from https://github.com/${REPO}/releases"
    fi
}

main "$@"
