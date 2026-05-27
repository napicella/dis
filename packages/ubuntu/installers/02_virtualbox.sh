### -- Manifest
### provides: common/virtualbox
### depends_on: []
### distro: [ubuntu]
### -- End


if command -v virtualbox &> /dev/null
then
    echo "virtualbox is installed"
    exit 0
fi

# Install virtual box
sudo apt install -y build-essential dkms linux-headers-$(uname -r)
sudo apt install -y virtualbox-7.0