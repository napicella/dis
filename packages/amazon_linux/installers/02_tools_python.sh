### -- Manifest
### provides: common/python
### depends_on: [common/brew]
### distro: [amazon_linux]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  echo "Installing Python via brew"
  brew install python
fi
