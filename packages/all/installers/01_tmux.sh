### -- Manifest
### provides: common/tmux
### depends_on: [common/os-libs]
### distro: [ubuntu]
### -- End

source $DIS_BINDING

sudo apt install -y tmux

# copy tmux config in the HOME dir
cp $DIS_CONFIG_FOLDER/.tmux.conf ~/