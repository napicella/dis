### -- Manifest
### provides: common/mise
### depends_on: [common/os-libs]
### distro: [ubuntu]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  if command -v mise &> /dev/null; then
    echo "mise is installed"
    exit 0
  fi

  # Install mise for managing multiple versions of languages. See https://mise.jdx.dev/
  # sudo apt update -y && sudo apt install -y gpg sudo wget curl
  sudo install -dm 755 /etc/apt/keyrings
  wget -qO - https://mise.jdx.dev/gpg-key.pub | gpg --dearmor | sudo tee /etc/apt/keyrings/mise-archive-keyring.gpg 1>/dev/null
  echo "deb [signed-by=/etc/apt/keyrings/mise-archive-keyring.gpg arch=amd64] https://mise.jdx.dev/deb stable main" | sudo tee /etc/apt/sources.list.d/mise.list
  sudo apt update -y
  sudo apt install -y mise

  # apt installs the mise cli under /usr/bin, so no need to add that to the path.
  dis tools add-rc-path --name 'Mise path' --content 'export PATH="$HOME/.local/share/mise/shims:$PATH"'

  dis tools add-rc-init \
    --name 'Mise activate' \
    --content 'eval "$(mise activate bash)"'
fi
