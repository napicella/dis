### -- Manifest
### provides: gui/ulauncher
### depends_on: []
### distro: [ubuntu]
### -- End

# https://github.com/Ulauncher/Ulauncher
# It's the only launcher that seems to work with Wayland and Gnome. 
# Just need to set the shortcut to open the window from the ubuntu keyboard shortcut instead of the ulancher preferences.
# What did not work: dmenu_run, bemnu_run, rofi.
#
# Recommended ulauncher extensions to install: 
# - https://ext.ulauncher.io/-/github-claudiosanches-ulauncher-window-switcher
# - https://ext.ulauncher.io/-/github-rlvendramini-ulauncher-gitmoji-ext


if [[ "$XDG_SESSION_TYPE" == "tty" ]]; then
    echo "GUI install not available on tty session type"
    exit 0
fi

if command -v ulauncher &> /dev/null
then
    echo "ulauncher is installed"
    exit 0
fi

sudo add-apt-repository universe -y
sudo add-apt-repository ppa:agornostal/ulauncher -y
sudo apt update -y
sudo apt install -y ulauncher

# Start ulauncher to have it populate config before we overwrite
mkdir -p ~/.config/autostart/
cp $DIS_CONFIG_FOLDER/ulauncher.desktop ~/.config/autostart/ulauncher.desktop
gtk-launch ulauncher.desktop >/dev/null 2>&1
sleep 2 # ensure enough time for ulauncher to set defaults
cp $DIS_CONFIG_FOLDER/ulauncher.json ~/.config/ulauncher/settings.json

# Install community themes: https://docs.ulauncher.io/en/latest/themes/themes.html
mkdir -p ~/.config/ulauncher/user-themes
git clone https://github.com/LucianoBigliazzi/ulauncher-nord.git ~/.config/ulauncher/user-themes/ulauncher-nord
git clone https://github.com/sociale11/ul-cosmo ~/.config/ulauncher/user-themes/ul-cosmo
git clone https://github.com/hmwassim/WhiteSur-Nord-ulauncher.git ~/.config/ulauncher/user-themes/WhiteSur-Nord-ulauncher
git clone https://github.com/SirHades696/TokyoNight-Ulauncher-Theme /tmp/TokyoNight-Ulauncher-Theme && cp -r /tmp/TokyoNight-Ulauncher-Theme/TokyoNight ~/.config/ulauncher/user-themes/
mkdir -p ~/.config/ulauncher/user-themes/Viridian git clone https://github.com/arthurrio/ulauncher-viridian-theme ~/.config/ulauncher/user-themes/Viridian
git clone https://github.com/napicella/Matcha-Dark-Aliz-ulauncher.git ~/.config/ulauncher/user-themes/Matcha-Dark-Aliz-ulauncher
