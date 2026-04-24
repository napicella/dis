### -- Manifest
### provides: common/thefuck
### depends_on: []
### distro: [ubuntu]
### -- End
# Install thefuck: https://github.com/nvbn/thefuck

source $DIS_BINDING

sudo apt -y install thefuck

# On Debian-based distros, manual activation is required. 
# The following adds the activation in the bash init which is included in bashrc

IMPORT_LINES='if command -v thefuck &> /dev/null
then
  eval $(thefuck --alias please)
  eval $(thefuck --alias fuck)
fi'


bashrc_init_add "TheFuck" "$IMPORT_LINES"