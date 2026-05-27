### -- Manifest
### provides: common/tldr
### depends_on: [common/node]
### distro: [all]
### -- End


# Install tldr: https://github.com/tldr-pages/tldr
npm install -g tldr
# copy theme config file to HOME
cp $DIS_CONFIG_FOLDER/.tldrrc ~/