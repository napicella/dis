### -- Manifest
### provides: common/node
### depends_on: [common/brew]
### distro: [amazon_linux]
### -- End

source $DIS_BINDING

echo "Installing Node.js via brew"
brew install node
