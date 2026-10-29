### -- Manifest
### provides: common/btop
### depends_on: [common/os-libs]
### distro: [ubuntu]
### -- End

# https://github.com/aristocratos/btop - resource monitor, from Ubuntu's universe repo.
if [[ -n "${DIS_INSTALL:-}" ]]; then
  sudo apt install -y btop
fi
