### -- Manifest
### provides: common/starship
### depends_on: []
### distro: [all]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  # Use this method only if the Ubuntu version is earlier than 25.04.
  # Ubuntu 25.04 and later provide the package in the Ubuntu repository.
  # https://starship.rs/guide/
  VER=$(. /etc/os-release && echo $VERSION_ID | sed 's/\.//')
  if (( $VER >= 2504 )); then
    sudo apt -y install starship
  else
    # not ubuntu or ubuntu earlier than 25.04
    curl -sS https://starship.rs/install.sh | sh
  fi

  # Wire starship into the shell RC (one-time setup).
  dis tools add-rc-init \
    --name 'Starship' \
    --content $'if [[ $- == *i* ]] && [[ ${TERM:-} != "dumb" ]] && command -v starship &> /dev/null; then
  eval "$(starship init bash)"
fi'
fi

# Config: always re-deploy the starship config file.
mkdir -p ~/.config/
cp $DIS_CONFIG_FOLDER/starship/starship.toml ~/.config/
