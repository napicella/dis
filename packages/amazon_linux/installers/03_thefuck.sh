### -- Manifest
### provides: common/thefuck
### depends_on: [common/brew]
### distro: [amazon_linux]
### -- End

# Install thefuck: https://github.com/nvbn/thefuck
# Amazon Linux has no package for it, so it comes from brew, like node and python.
if [[ -n "${DIS_INSTALL:-}" ]]; then
  brew install thefuck
fi

# thefuck needs its aliases set up by hand; the bash init is included in bashrc.
dis tools add-rc-init \
  --name 'TheFuck' \
  --content 'if command -v thefuck &> /dev/null
then
  eval $(thefuck --alias please)
  eval $(thefuck --alias fuck)
fi'
