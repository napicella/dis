### -- Manifest
### provides: gui/gnome-tweak-tool
### depends_on: []
### distro: [ubuntu]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  sudo apt install -y gnome-tweak-tool | sudo apt install -y gnome-tweaks
fi
