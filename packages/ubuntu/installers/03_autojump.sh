### -- Manifest
### provides: common/autojump
### depends_on: [common/os-libs]
### distro: [ubuntu]
### -- End

source $DIS_BINDING

# Install autojump: https://www.linode.com/docs/guides/faster-file-navigation-with-autojump
sudo apt install -y autojump

# On Debian-based distros, manual activation is required. 
# The following adds the activation in the bash init which is included in bashrc

bashrc_init_add "Autojump" 'if [ -f /usr/share/autojump/autojump.sh ]; then
  . /usr/share/autojump/autojump.sh
fi'

