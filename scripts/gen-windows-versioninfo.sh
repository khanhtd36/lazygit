#!/bin/sh
# Embeds a Windows PE file-version resource into the windows/$GOARCH build,
# so tools like `Get-Command`/Explorer's Properties tab show the real
# version instead of 0.0.0.0. Called as a goreleaser per-build pre-hook;
# goreleaser sets GOOS/GOARCH for the build it's about to run, so this is a
# no-op for every non-windows target.
set -eu

if [ "${GOOS:-}" != "windows" ]; then
    exit 0
fi

VERSION="$1"
MAJOR="$2"
MINOR="$3"
PATCH="$4"

CONFIG="$(mktemp)"
trap 'rm -f "$CONFIG"' EXIT

cat > "$CONFIG" <<EOF
{
  "FixedFileInfo": {
    "FileVersion": { "Major": ${MAJOR}, "Minor": ${MINOR}, "Patch": ${PATCH}, "Build": 0 },
    "ProductVersion": { "Major": ${MAJOR}, "Minor": ${MINOR}, "Patch": ${PATCH}, "Build": 0 }
  },
  "StringFileInfo": {
    "FileDescription": "lazygit - a terminal UI for git",
    "ProductName": "lazygit",
    "ProductVersion": "${VERSION}",
    "FileVersion": "${VERSION}",
    "CompanyName": "khanhtd36",
    "LegalCopyright": "MIT License"
  },
  "VarFileInfo": { "Translation": { "LangID": "0409", "CharsetID": "04B0" } }
}
EOF

go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.4.1 \
    -o "resource_windows_${GOARCH}.syso" \
    "$CONFIG"
