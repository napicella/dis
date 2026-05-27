### -- Manifest
### provides: common/mise
### depends_on: [common/os-libs]
### distro: [amazon_linux]
### -- End

if command -v mise &> /dev/null
then
    echo "mise is installed"
    exit 0
fi

# Install mise for managing multiple versions of languages. See https://mise.jdx.dev/
curl https://mise.run | sh
dis tools add-rc-path --name 'Mise path' --content 'export PATH="$HOME/.local/bin:$PATH"'
