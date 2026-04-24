#!/usr/bin/env bash
set -euo pipefail

# dis installer
# Usage: curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/install.sh | GITHUB_TOKEN=<token> bash

REPO="napicella/dis"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
BINARY_NAME="dis"

# --- Require token for private repo ---
if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "Error: GITHUB_TOKEN is required."
  echo "Usage: curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/install.sh | GITHUB_TOKEN=<token> bash"
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
LATEST_URL="https://api.github.com/repos/${REPO}/releases/latest"
TAG="$(curl -fsSL -H "Authorization: Bearer ${GITHUB_TOKEN}" -H "Accept: application/vnd.github+json" \
  "${LATEST_URL}" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"

if [[ -z "${TAG}" ]]; then
  echo "Error: could not determine latest release tag."
  exit 1
fi

echo "==> Installing dis ${TAG} (${OS}/${ARCH})"

# --- Download and extract ---
TARBALL="${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${TARBALL}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

curl -fsSL \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "Accept: application/octet-stream" \
  -L "${DOWNLOAD_URL}" \
  -o "${TMP_DIR}/${TARBALL}"

tar -xzf "${TMP_DIR}/${TARBALL}" -C "${TMP_DIR}"

# --- Install binary ---
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
  echo "==> Restart your shell or run: export PATH=\"${INSTALL_DIR}:\${PATH}\""
fi

echo "==> Done. Run 'dis --help' to get started."
