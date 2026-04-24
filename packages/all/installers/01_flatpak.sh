### -- Manifest
### provides: common/flatpak
### depends_on: [common/os-libs]
### distro: [ubuntu]
### -- End

source $DIS_BINDING

if command -v flatpak &> /dev/null
then
    echo "flatpak is installed"
    exit 0
fi

sudo apt install -y flatpak
sudo apt install -y gnome-software-plugin-flatpak
sudo flatpak remote-add --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo