### -- Manifest
### provides: common/mise
### depends_on: [common/os-libs]
### distro: [amazon_linux]
### -- End

source $DIS_BINDING

if command -v mise &> /dev/null
then
    echo "mise is installed"
    exit 0
fi

# Install mise for managing multiple versions of languages. See https://mise.jdx.dev/
curl https://mise.run | sh
bashrc_path_add "Mise path" 'export PATH="$HOME/.local/bin:$PATH"'