### -- Manifest
### provides: common/bash-config
### depends_on: []
### distro: [all]
### -- End

source $DIS_BINDING

# Copy bash_config.sh to ~/rc/
mkdir -p ~/rc
cp "$DIS_CONFIG_FOLDER/bash_config.sh" ~/rc/bash_config.sh

# Idempotently add the source line to ~/.bashrc
if ! grep -qF '# dis: source bash config' ~/.bashrc; then
    echo '' >> ~/.bashrc
    echo '# dis: source bash config' >> ~/.bashrc
    echo '[ -f ~/rc/bash_config.sh ] && source ~/rc/bash_config.sh' >> ~/.bashrc
fi
