### -- Manifest
### provides: common/starship
### depends_on: []
### distro: [ubuntu]
### -- End

sudo apt -y install starship
mkdir -p ~/.config/
if [ ! ~/.config/starship.toml ]; then
  cp $DIS_CONFIG/starship/starship.toml ~/.config/
fi

dis tools add-rc-init \
  --name 'Starship' \
  --content 'eval "$(starship init bash)"'
