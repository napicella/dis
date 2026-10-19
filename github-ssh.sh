#!/usr/bin/env bash
set -euo pipefail

# GitHub SSH key setup for a new host
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/github-ssh.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/github-ssh.sh | bash -s -- --key ~/.ssh/my_github
#
# Gives this host its own GitHub key, so `dis pull` can clone private repos over
# SSH. Safe to re-run: it stops as soon as GitHub accepts the host's key, never
# overwrites a key, and only edits the ~/.ssh/config block it wrote.
#   1. adds GitHub's published host keys to ~/.ssh/known_hosts
#   2. adds a "Host github.com" block to ~/.ssh/config that uses the key
#   3. creates the key with ssh-keygen (it asks for a passphrase), unless it exists
#   4. prints the public key and waits for you to add it on GitHub
#   5. checks that GitHub accepts it

KEY="${HOME}/.ssh/github_ed25519"
SSH_DIR="${HOME}/.ssh"
CONFIG="${SSH_DIR}/config"
KNOWN_HOSTS="${SSH_DIR}/known_hosts"
BEGIN_MARK="# BEGIN dis github-ssh"
END_MARK="# END dis github-ssh"
# Not $HOSTNAME: /etc/profile on Amazon Linux overwrites it, empty without hostnamectl.
HOST="$(uname -n)"
HOST="${HOST%%.*}"

usage() {
  echo "Usage: github-ssh.sh [--key PATH]"
  echo "  --key PATH  private key to create or reuse (default: ~/.ssh/github_ed25519)"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --key) KEY="${2:?--key needs a path}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Error: unknown argument: $1"; usage; exit 1 ;;
  esac
done
KEY="${KEY/#\~/${HOME}}"

for cmd in ssh ssh-keygen curl; do
  if ! command -v "${cmd}" >/dev/null; then
    echo "Error: ${cmd} is required (install openssh-client and curl)."
    exit 1
  fi
done

# check_github succeeds when GitHub accepts this host's own key, and sets
# GITHUB_USER. Agent keys are ignored (e.g. one forwarded from a laptop): the
# point is that the host can clone on its own. SSH_OUT keeps ssh's output.
check_github() {
  local re='Hi ([^!]+)!.*successfully authenticated'
  SSH_OUT="$(env -u SSH_AUTH_SOCK ssh -T -o ConnectTimeout=10 "$@" git@github.com 2>&1 </dev/null || true)"
  [[ "${SSH_OUT}" =~ ${re} ]] || return 1
  GITHUB_USER="${BASH_REMATCH[1]}"
}

# Pins the keys GitHub publishes over HTTPS, so the first clone neither asks to
# confirm github.com's fingerprint nor trusts whatever answers on port 22.
add_github_host_keys() {
  local keys key found added=0
  if ! keys="$(curl -fsSL https://api.github.com/meta \
      | grep -oE '"(ssh-ed25519|ecdsa-sha2-nistp256|ssh-rsa) [A-Za-z0-9+/=]+"' | tr -d '"')"; then
    echo "==> Could not fetch GitHub's host keys; ssh will ask you to confirm github.com's fingerprint"
    return 0
  fi
  found="$(ssh-keygen -F github.com -f "${KNOWN_HOSTS}" 2>/dev/null || true)"
  while read -r key; do
    if [[ "${found}" != *"${key}"* ]]; then
      echo "github.com ${key}" >> "${KNOWN_HOSTS}"
      added=1
    fi
  done <<< "${keys}"
  if (( added )); then
    echo "==> Added GitHub's host keys to ${KNOWN_HOSTS}"
  fi
}

# ssh_uses_key succeeds when ssh's effective config for github.com offers KEY.
ssh_uses_key() {
  local f
  while read -r _ f; do
    [[ "${f/#\~/${HOME}}" == "${KEY}" ]] && return 0
  done < <(ssh -G github.com 2>/dev/null | grep -i '^identityfile ')
  return 1
}

# The key's path as written in ~/.ssh/config: ~/... when it's under $HOME.
config_path() {
  if [[ "$1" == "${HOME}/"* ]]; then echo "~/${1#"${HOME}/"}"; else echo "$1"; fi
}

configure_ssh() {
  touch "${CONFIG}"
  chmod 600 "${CONFIG}"
  if ssh_uses_key; then
    echo "==> ${CONFIG} already uses $(config_path "${KEY}") for github.com"
    return 0
  fi
  if grep -qxF "${BEGIN_MARK}" "${CONFIG}"; then
    # Our block, for another key: drop it, it's added again below.
    local kept
    kept="$(awk -v b="${BEGIN_MARK}" -v e="${END_MARK}" '$0==b{skip=1} !skip{print} $0==e{skip=0}' "${CONFIG}")"
    if [[ -n "${kept//[[:space:]]/}" ]]; then printf '%s\n' "${kept}" > "${CONFIG}"; else : > "${CONFIG}"; fi
  elif grep -qiE '^[[:space:]]*Host[[:space:]]+(.*[[:space:]])?github\.com([[:space:]]|$)' "${CONFIG}"; then
    echo "Error: ${CONFIG} already has a Host github.com block, and it doesn't use $(config_path "${KEY}")."
    echo "       Re-run with --key <the IdentityFile of that block>, or remove the block."
    exit 1
  fi
  local sep=''
  if [[ -s "${CONFIG}" ]]; then sep=$'\n'; fi
  printf '%s%s\nHost github.com\n  IdentityFile %s\n  IdentitiesOnly yes\n%s\n' \
    "${sep}" "${BEGIN_MARK}" "$(config_path "${KEY}")" "${END_MARK}" >> "${CONFIG}"
  echo "==> Added a github.com block to ${CONFIG}"
}

create_key() {
  if [[ -f "${KEY}" ]]; then
    echo "==> Using the existing key ${KEY}"
  else
    echo "==> Creating ${KEY} (press Enter for no passphrase)"
    ssh-keygen -t ed25519 -f "${KEY}" -C "${USER:-$(id -un)}@${HOST}" </dev/tty
  fi
  if [[ ! -f "${KEY}.pub" ]]; then
    ssh-keygen -y -f "${KEY}" > "${KEY}.pub" </dev/tty
  fi
}

mkdir -p "${SSH_DIR}"
chmod 700 "${SSH_DIR}"
add_github_host_keys

if check_github -o BatchMode=yes; then
  echo "==> Already set up: GitHub accepts this host's key as ${GITHUB_USER}"
  exit 0
fi

if ! { : </dev/tty; } 2>/dev/null; then
  echo "Error: no terminal. It's needed for the key passphrase and to wait while you add the key on GitHub."
  exit 1
fi

# Before creating the key: fails early when another block owns github.com.
configure_ssh
create_key

echo
echo "==> Add this public key on GitHub: https://github.com/settings/ssh/new"
echo "    Title: ${HOST}  Key type: Authentication Key"
echo "    (If it's already there, just press Enter.)"
echo
cat "${KEY}.pub"
echo
while true; do
  read -r -p "Press Enter once it's added (Ctrl-C to quit)... " </dev/tty
  if check_github; then
    echo "==> Done: GitHub accepts this host's key as ${GITHUB_USER}"
    exit 0
  fi
  echo "GitHub doesn't accept $(config_path "${KEY}") yet. ssh said:"
  echo "${SSH_OUT}" | tail -n 3 | sed 's/^/    /'
done
