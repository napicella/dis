### -- Manifest
### provides: common/lazydocker
### depends_on: [common/os-libs]
### distro: [all]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  # The GitHub API allows 60 unauthenticated requests per hour: fail with a
  # message instead of exiting silently when it doesn't answer.
  LAZYDOCKER_VERSION=$(curl -fsS "https://api.github.com/repos/jesseduffield/lazydocker/releases/latest" | grep -Po '"tag_name": "v\K[^"]*' || true)
  if [[ -z "$LAZYDOCKER_VERSION" ]]; then
    echo "lazydocker: could not get the latest version from the GitHub API" >&2
    exit 1
  fi

  tmp_dir=$(mktemp -d)
  trap 'rm -rf "$tmp_dir"' EXIT
  curl -fsSLo "$tmp_dir/lazydocker.tar.gz" "https://github.com/jesseduffield/lazydocker/releases/latest/download/lazydocker_${LAZYDOCKER_VERSION}_Linux_x86_64.tar.gz"
  tar -xf "$tmp_dir/lazydocker.tar.gz" -C "$tmp_dir" lazydocker
  sudo install "$tmp_dir/lazydocker" /usr/local/bin
fi