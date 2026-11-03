### -- Manifest
### provides: common/gum
### depends_on: [common/os-libs]
### distro: [amazon_linux]
### -- End

# https://github.com/charmbracelet/gum - from Charm's yum repo, as the ubuntu variant uses
# its apt repo. A system package lands in /usr/bin, so it is on PATH even for shells that
# don't read the rc files, like the herdr shortcut popup.
if [[ -n "${DIS_INSTALL:-}" ]]; then
  if command -v gum &> /dev/null; then
      echo "gum is installed"
      exit 0
  fi

  sudo tee /etc/yum.repos.d/charm.repo > /dev/null <<'EOF'
[charm]
name=Charm
baseurl=https://repo.charm.sh/yum/
enabled=1
gpgcheck=1
gpgkey=https://repo.charm.sh/yum/gpg.key
EOF
  # Drop the repo again if yum can't use it, so later installers can still run yum.
  if ! { sudo rpm --import https://repo.charm.sh/yum/gpg.key && sudo yum install -y gum; }; then
    sudo rm -f /etc/yum.repos.d/charm.repo
    exit 1
  fi
fi
