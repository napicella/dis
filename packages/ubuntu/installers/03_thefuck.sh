### -- Manifest
### provides: common/thefuck
### depends_on: []
### distro: [ubuntu]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  # Install thefuck: https://github.com/nvbn/thefuck
  sudo apt -y install thefuck

  # On Debian-based distros, manual activation is required.
  # The following adds the activation in the bash init which is included in bashrc.
  dis tools add-rc-init \
    --name 'TheFuck' \
    --content 'if command -v thefuck &> /dev/null
then
  eval $(thefuck --alias please)
  eval $(thefuck --alias fuck)
fi'
fi
