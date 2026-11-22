#!/usr/bin/env bash
set -euo pipefail

# dis installer
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/install.sh | bash -s -- <distro-repo-url> <distro-file>
# With a distro repo and the distro file in it, it also sets up this host: a
# GitHub SSH key (github-ssh.sh), then dis pull and dis install.

REPO="napicella/dis"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
BINARY_NAME="dis"

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
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/${REPO}/releases/tags/${TAG}")"

ASSET_ID="$(echo "${RELEASE_JSON}" \
  | jq -r --arg name "${BINARY_TARBALL}" '.assets[] | select(.name == $name) | .id')"

if [[ -z "${ASSET_ID}" ]]; then
  echo "Error: could not find asset '${BINARY_TARBALL}' in release ${TAG}."
  exit 1
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

curl -fsSL \
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

if [[ $# -ge 2 ]]; then  # install.sh <distro-repo-url> <distro-file>: set up this host
  # Touch 1: add the host's key on GitHub; it stops at once if GitHub accepts it.
  curl -fsSL "https://raw.githubusercontent.com/${REPO}/main/github-ssh.sh" | bash
  # With `curl ... | bash`, bash reads this script from its stdin, the pipe.
  # dis and its installers would get that pipe as their stdin too, already at
  # its end, so an installer that asks a question (read -p) would read nothing
  # and fail. So each dis command below reads from the terminal instead.
  # This is done per command, not once with `exec </dev/tty`: that would also
  # switch bash's own stdin, and bash would read the rest of this script from
  # the keyboard.
  tty_in=/dev/stdin
  if { : </dev/tty; } 2>/dev/null; then tty_in=/dev/tty; fi
  "${INSTALL_DIR}/${BINARY_NAME}" pull "$1" --distro "$2" <"${tty_in}"
  # Touch 2, only if an installer needs a secret and ~/.config/dis/bws-token is missing.
  "${INSTALL_DIR}/${BINARY_NAME}" install <"${tty_in}"
else
  echo "==> Done. Next, fetch the repos of your distro and install it:"
  echo "      dis pull <git-url of the repo with your distro> --distro <path/to/distro.yml>"
  echo "      dis install"
fi
