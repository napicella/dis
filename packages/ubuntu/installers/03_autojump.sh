### -- Manifest
### provides: common/autojump
### depends_on: [common/os-libs]
### distro: [ubuntu]
### -- End

# Install autojump: https://www.linode.com/docs/guides/faster-file-navigation-with-autojump
sudo apt install -y autojump

# On Debian-based distros, manual activation is required. 
# The following adds the activation in the bash init which is included in bashrc
dis tools add-rc-init \
  --name 'Autojump' \
  --content 'if [ -f /usr/share/autojump/autojump.sh ]; then
  . /usr/share/autojump/autojump.sh
fi'
