#!/usr/bin/env bash
set -euo pipefail

# dis installer
# Usage: curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/install.sh | GITHUB_TOKEN=<token> bash

REPO="napicella/dis"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
PACKAGES_DIR="${PACKAGES_DIR:-${HOME}/.local/share/dis/packages}"
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

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

gh_download() {
  local filename="$1"
  # For private repos we must resolve the asset ID via the API first.
  local release_url="https://api.github.com/repos/${REPO}/releases/tags/${TAG}"
  local asset_id
  asset_id="$(curl -fsSL \
    -H "Authorization: Bearer ${GITHUB_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    "${release_url}" \
    | grep -A2 "\"name\": \"${filename}\"" \
    | grep '"id":' \
    | head -1 \
    | sed 's/.*"id": *\([0-9]*\).*/\1/')"

  if [[ -z "${asset_id}" ]]; then
    echo "Error: could not find asset '${filename}' in release ${TAG}."
    exit 1
  fi

  curl -fsSL \
    -H "Authorization: Bearer ${GITHUB_TOKEN}" \
    -H "Accept: application/octet-stream" \
    "https://api.github.com/repos/${REPO}/releases/assets/${asset_id}" \
    -o "${TMP_DIR}/${filename}"
}

# --- Install binary ---
BINARY_TARBALL="${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
gh_download "${BINARY_TARBALL}"
tar -xzf "${TMP_DIR}/${BINARY_TARBALL}" -C "${TMP_DIR}"
mkdir -p "${INSTALL_DIR}"
install -m 755 "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
echo "==> Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"

# --- Install packages ---
PACKAGES_TARBALL="${BINARY_NAME}_packages.tar.gz"
gh_download "${PACKAGES_TARBALL}"
# The tarball contains packages/ at the root; extract then move contents to PACKAGES_DIR.
tar -xzf "${TMP_DIR}/${PACKAGES_TARBALL}" -C "${TMP_DIR}"
mkdir -p "${PACKAGES_DIR}"
cp -r "${TMP_DIR}/packages/." "${PACKAGES_DIR}/"
echo "==> Installed packages to ${PACKAGES_DIR}"

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
