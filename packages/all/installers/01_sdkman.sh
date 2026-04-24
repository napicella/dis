### -- Manifest
### provides: common/sdkman
### depends_on: []
### distro: [all]
### -- End

source $DIS_BINDING

# Install sdkman: https://sdkman.io/install

if command -v sdk &> /dev/null
then
    echo "sdkman is installed"
    exit 0
fi

curl -s "https://get.sdkman.io" | bash

bashrc_init_add "Sdkman" 'export SDKMAN_DIR="$HOME/.sdkman"
[[ -s "$HOME/.sdkman/bin/sdkman-init.sh" ]] && source "$HOME/.sdkman/bin/sdkman-init.sh"
'