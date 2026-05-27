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