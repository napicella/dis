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

  code --install-extension golang.Go
fi

# Settings and keybindings come from VS Code Settings Sync.
echo "gui/vscode: Settings and keybindings come from VS Code Settings Sync. 
If not done yet, turn on Settings Sync (Accounts menu), with Settings and Keybindings ticked."
