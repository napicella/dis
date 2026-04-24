### -- Manifest
### provides: common/cheat
### depends_on: [common/languages]
### distro: [all]
### -- End
source $DIS_BINDING

 go install github.com/cheat/cheat/cmd/cheat@latest

echo "Rendering the cheat configuration"
config_path="$DIS_CONFIG_FOLDER/cheat/conf.yml"
envsubst < "$config_path" > /tmp/cheat_rendered.yml

echo "Installing the cheat configuration"
mkdir -p $HOME/.config/cheat
mv /tmp/cheat_rendered.yml $HOME/.config/cheat/conf.yml

echo "Done"
