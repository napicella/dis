### -- Manifest
### provides: common/ulauncher
### depends_on: []
### distro: [ubuntu]
### -- End
# https://github.com/Ulauncher/Ulauncher
# It's the only launcher that seems to work with Wayland and Gnome. 
# Just need to set the shortcut to open the window from the ubuntu keyboard shortcut instead of the ulancher preferences.
# What did not work: dmenu_run, bemnu_run, rofi.

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