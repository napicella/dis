### -- Manifest
### provides: common/tldr
### depends_on: [common/node]
### distro: [all]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  # Install tldr: https://github.com/tldr-pages/tldr
  npm install -g tldr
fi

# Config: always re-deploy the tldr config file.
cp $DIS_CONFIG_FOLDER/.tldrrc ~/
