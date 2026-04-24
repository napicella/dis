#!/usr/bin/env bash
set -euo pipefail

# dis installer
# Usage:
#   export GITHUB_TOKEN=ghp_...
#   curl -fsSL -H "Authorization: Bearer ${GITHUB_TOKEN}" \
#     https://raw.githubusercontent.com/napicella/dis/main/install.sh \
#     | GITHUB_TOKEN="${GITHUB_TOKEN}" bash

REPO="napicella/dis"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
BINARY_NAME="dis"

# --- Require token ---
if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "Error: GITHUB_TOKEN is required."
  exit 1
fi

# --- Detect OS and arch ---
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Error: unsupported architecture: ${ARCH}"
    exit 1
    ;;
esac

# --- Fetch latest release tag ---
TAG="$(curl -fsSL \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"

if [[ -z "${TAG}" ]]; then
  echo "Error: could not determine latest release tag."
  exit 1
fi

echo "==> Installing dis ${TAG} (${OS}/${ARCH})"

# --- Download binary via asset ID ---
BINARY_TARBALL="${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
RELEASE_JSON="$(curl -fsSL \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/${REPO}/releases/tags/${TAG}")"

ASSET_ID="$(echo "${RELEASE_JSON}" \
  | grep -B1 "\"name\": \"${BINARY_TARBALL}\"" \
  | grep '"id":' | tail -1 \
  | sed 's/.*"id": *\([0-9]*\).*/\1/')"

# Fallback: use python if grep approach fails (python is more available than jq)
if [[ -z "${ASSET_ID}" ]] && command -v python3 &>/dev/null; then
  ASSET_ID="$(echo "${RELEASE_JSON}" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for a in data['assets']:
    if a['name'] == '${BINARY_TARBALL}':
        print(a['id'])
        break
")"
fi

if [[ -z "${ASSET_ID}" ]]; then
  echo "Error: could not find asset '${BINARY_TARBALL}' in release ${TAG}."
  exit 1
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

curl -fsSL \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "Accept: application/octet-stream" \
  "https://api.github.com/repos/${REPO}/releases/assets/${ASSET_ID}" \
  -o "${TMP_DIR}/${BINARY_TARBALL}"

tar -xzf "${TMP_DIR}/${BINARY_TARBALL}" -C "${TMP_DIR}"
mkdir -p "${INSTALL_DIR}"
install -m 755 "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
echo "==> Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"

# --- Add to PATH if needed ---
add_to_path() {
  local rc_file="$1"
  local line='export PATH="${HOME}/.local/bin:${PATH}"'
  if [[ -f "${rc_file}" ]] && ! grep -qF '.local/bin' "${rc_file}"; then
    echo "" >> "${rc_file}"
    echo "# Added by dis installer" >> "${rc_file}"
    echo "${line}" >> "${rc_file}"
    echo "==> Added ${INSTALL_DIR} to PATH in ${rc_file}"
  fi
}

if [[ ":${PATH}:" != *":${INSTALL_DIR}:"* ]]; then
  add_to_path "${HOME}/.bashrc"
  add_to_path "${HOME}/.zshrc"
  export PATH="${INSTALL_DIR}:${PATH}"
fi

# --- Sync common packages using the binary ---
echo "==> Syncing common packages..."
"${INSTALL_DIR}/${BINARY_NAME}" sync --token "${GITHUB_TOKEN}"

echo "==> Done. Run 'dis --help' to get started."
