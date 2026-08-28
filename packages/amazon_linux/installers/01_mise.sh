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
# Install it under /usr/bin like package managers do
curl https://mise.run -o /tmp/install-mise.sh && chmod +x /tmp/install-mise.sh
sudo MISE_INSTALL_PATH=/usr/bin/mise /tmp/install-mise.sh
