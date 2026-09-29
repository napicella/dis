### -- Manifest
### provides: common/gum
### depends_on: [common/os-libs]
### distro: [ubuntu]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  if command -v gum &> /dev/null; then
      echo "gum is installed"
      exit 0
  fi

  # Download the key to a file first: in a pipeline a failed download goes unnoticed and
  # leaves an empty keyring behind a source that breaks every later 'apt update'.
  key=$(mktemp)
  curl -fsSL https://repo.charm.sh/apt/gpg.key -o "$key"
  [[ -s "$key" ]] || { echo "Downloaded signing key from https://repo.charm.sh/apt/gpg.key is empty"; exit 1; }
  sudo install -dm 755 /etc/apt/keyrings
  sudo gpg --batch --yes --dearmor -o /etc/apt/keyrings/charm.gpg "$key"
  rm -f "$key"
  echo "deb [signed-by=/etc/apt/keyrings/charm.gpg] https://repo.charm.sh/apt/ * *" | sudo tee /etc/apt/sources.list.d/charm.list >/dev/null
  # Drop the source again if apt can't use it, so later installers can still run apt update.
  if ! sudo apt update -y; then
    sudo rm -f /etc/apt/sources.list.d/charm.list
    exit 1
  fi
  sudo apt install -y gum
fi
