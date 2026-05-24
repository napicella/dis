### -- Manifest
### provides: gui/gnome-theme
### depends_on: [gui/gnome-extensions]
### distro: [ubuntu]
### -- End

source $DIS_BINDING

OMAKUB_THEME_COLOR="purple"
OMAKUB_THEME_BACKGROUND="80s-retro-tropical-sunset-by-freepik.jpg"

gsettings set org.gnome.desktop.interface color-scheme 'prefer-dark'
gsettings set org.gnome.desktop.interface cursor-theme 'Yaru'
gsettings set org.gnome.desktop.interface gtk-theme "Yaru-$OMAKUB_THEME_COLOR-dark"
gsettings set org.gnome.desktop.interface icon-theme "Yaru-$OMAKUB_THEME_COLOR"

BACKGROUND_ORG_PATH="$DIS_CONFIG_FOLDER/backgrounds/$OMAKUB_THEME_BACKGROUND"
BACKGROUND_DEST_DIR="$HOME/.local/share/backgrounds"
BACKGROUND_DEST_PATH="$BACKGROUND_DEST_DIR/$OMAKUB_THEME_BACKGROUND"

if [ ! -d "$BACKGROUND_DEST_DIR" ]; then mkdir -p "$BACKGROUND_DEST_DIR"; fi

[ ! -f $BACKGROUND_DEST_PATH ] && cp $BACKGROUND_ORG_PATH $BACKGROUND_DEST_PATH
gsettings set org.gnome.desktop.background picture-uri $BACKGROUND_DEST_PATH
gsettings set org.gnome.desktop.background picture-uri-dark $BACKGROUND_DEST_PATH
gsettings set org.gnome.desktop.background picture-options 'zoom'

# note: default backgrounds are in /usr/share/backgrounds