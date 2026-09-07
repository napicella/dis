### -- Manifest
### provides: gui/vscode
### depends_on: [common/os-libs]
### distro: [ubuntu]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  if ! command -v code &> /dev/null; then
    cd /tmp

    # Note, you can pin the version as in the example below:
    # wget -O code.deb 'https://update.code.visualstudio.com/1.93.1/linux-deb-x64/stable'
    wget -O code.deb 'https://update.code.visualstudio.com/latest/linux-deb-x64/stable'
    sudo DEBIAN_FRONTEND=noninteractive apt install -y ./code.deb
    rm code.deb
    cd -
  fi

  # Install default supported themes
  code --install-extension enkia.tokyo-night
  code --install-extension golang.Go
fi

# Config: always re-deploy VS Code settings and keybindings.
mkdir -p ~/.config/Code/User
cp $DIS_CONFIG_FOLDER/vscode/vs-code-settings.json $HOME/.config/Code/User/settings.json
cp $DIS_CONFIG_FOLDER/vscode/keybindings.json $HOME/.config/Code/User/keybindings.json
