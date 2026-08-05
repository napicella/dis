### -- Manifest
### provides: common/starship
### depends_on: []
### distro: [all]
### -- End

mkdir -p ~/.config/
cp $DIS_CONFIG_FOLDER/starship/starship.toml ~/.config/

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

dis tools add-rc-init \
  --name 'Starship' \
  --content $'if [[ $- == *i* ]] && [[ ${TERM:-} != "dumb" ]] && command -v starship &> /dev/null; then
  eval "$(starship init bash)"
fi'
