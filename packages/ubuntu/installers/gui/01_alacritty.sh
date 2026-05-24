### -- Manifest
### provides: gui/alacritty
### depends_on: []
### distro: [ubuntu]
### -- End
source $DIS_BINDING

# Alacritty is a GPU-powered and highly extensible terminal. See https://alacritty.org/
sudo snap install alacritty --classic
mkdir -p ~/.config/alacritty
cp $DIS_CONFIG_FOLDER/alacritty/alacritty.toml ~/.config/alacritty/alacritty.toml
cp $DIS_CONFIG_FOLDER/themes/tokyo-night/alacritty.toml ~/.config/alacritty/theme.toml
cp $DIS_CONFIG_FOLDER/alacritty/fonts/CaskaydiaMono.toml ~/.config/alacritty/font.toml
cp $DIS_CONFIG_FOLDER/alacritty/font-size.toml ~/.config/alacritty/
cp $DIS_CONFIG_FOLDER/alacritty/pane.toml ~/.config/alacritty/
cp $DIS_CONFIG_FOLDER/alacritty/btop.toml ~/.config/alacritty/
cp $DIS_CONFIG_FOLDER/alacritty/defaults.toml ~/.config/alacritty/