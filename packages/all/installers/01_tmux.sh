### -- Manifest
### provides: common/tmux
### depends_on: [common/os-libs]
### distro: [ubuntu]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  sudo apt install -y tmux
fi

# Config: always re-deploy the tmux config file.
cp $DIS_CONFIG_FOLDER/.tmux.conf ~/
