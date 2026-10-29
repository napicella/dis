### -- Manifest
### provides: common/btop
### depends_on: [common/brew]
### distro: [amazon_linux]
### -- End

# https://github.com/aristocratos/btop - resource monitor. Amazon Linux has no btop
# package, so install it from brew.
if [[ -n "${DIS_INSTALL:-}" ]]; then
  brew install btop
fi
