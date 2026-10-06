### -- Manifest
### provides: common/bash-config
### depends_on: []
### distro: [all]
### -- End

# Copy bash_config.sh to ~/rc/
mkdir -p ~/rc
cp "$DIS_CONFIG_FOLDER/bash_config.sh" ~/rc/bash_config.sh

# Wire ~/.bashrc to source the bash config.
dis tools add-home-rc \
  --name 'bash config' \
  --content '[ -f ~/rc/bash_config.sh ] && source ~/rc/bash_config.sh;'

# Add ./local/bin to the path
dis tools add-rc-path \
  --name './local/bin' \
  --path '$HOME/.local/bin'

# Default terminal editor. The guards keep a value set earlier, so a package
# can override it.
dis tools add-rc-env \
  --name 'Editor default' \
  --content 'export EDITOR="${EDITOR:-vim}"
export VISUAL="${VISUAL:-$EDITOR}"'
