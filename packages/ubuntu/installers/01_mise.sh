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
  # Download the key to a file first: in a pipeline a failed download goes unnoticed and
  # leaves an empty keyring behind a source that breaks every later 'apt update'.
  key=$(mktemp)
  curl -fsSL https://mise.jdx.dev/gpg-key.pub -o "$key"
  [[ -s "$key" ]] || { echo "Downloaded signing key from https://mise.jdx.dev/gpg-key.pub is empty"; exit 1; }
  sudo install -dm 755 /etc/apt/keyrings
  sudo gpg --batch --yes --dearmor -o /etc/apt/keyrings/mise-archive-keyring.gpg "$key"
  rm -f "$key"
  echo "deb [signed-by=/etc/apt/keyrings/mise-archive-keyring.gpg arch=amd64] https://mise.jdx.dev/deb stable main" | sudo tee /etc/apt/sources.list.d/mise.list >/dev/null
  # Drop the source again if apt can't use it, so later installers can still run apt update.
  if ! sudo apt update -y; then
    sudo rm -f /etc/apt/sources.list.d/mise.list
    exit 1
  fi
  sudo apt install -y mise

  # apt installs the mise cli under /usr/bin, so no need to add that to the path.
  dis tools add-rc-path --name 'Mise path' --content 'export PATH="$HOME/.local/share/mise/shims:$PATH"'

  dis tools add-rc-init \
    --name 'Mise activate' \
    --content 'eval "$(mise activate bash)"'
fi
