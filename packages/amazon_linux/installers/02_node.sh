### -- Manifest
### provides: common/node
### depends_on: [common/brew]
### distro: [amazon_linux]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  echo "Installing Node.js via brew"
  brew install node
fi
