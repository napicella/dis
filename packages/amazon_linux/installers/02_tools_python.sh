### -- Manifest
### provides: common/python
### depends_on: [common/brew]
### distro: [amazon_linux]
### -- End

source $DIS_BINDING

echo "Installing Python via brew"
brew install python
