### -- Manifest
### provides: common/docker
### depends_on: []
### distro: [amazon_linux]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  # Some hosts come with Docker preinstalled and managed by their own tooling:
  # leave that one alone.
  if command -v docker &> /dev/null; then
    echo "docker is already installed"
    exit 0
  fi

  # Docker from the Amazon Linux repos (Amazon Linux 2 and 2023).
  sudo yum install -y docker

  # Give this user privileged Docker access (takes effect at the next login).
  sudo usermod -aG docker "${USER}"

  # Limit log size to avoid running out of disk
  echo '{"log-driver":"json-file","log-opts":{"max-size":"10m","max-file":"5"}}' | sudo tee /etc/docker/daemon.json

  # Start Docker now and at boot, only when systemd is running (not the case in
  # containers or WSL).
  if [[ -d /run/systemd/system ]]; then
    sudo systemctl enable --now docker
  fi
fi
