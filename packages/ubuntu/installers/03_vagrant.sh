### -- Manifest
### provides: common/vagrant
### depends_on: [common/os-libs]
### distro: [ubuntu]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  if command -v vagrant &> /dev/null; then
      echo "vagrant is installed"
      exit 0
  fi

  # Install vagrant
  # Download the key to a file first: in a pipeline a failed download goes unnoticed and
  # leaves an empty keyring behind a source that breaks every later 'apt update'.
  key=$(mktemp)
  curl -fsSL https://apt.releases.hashicorp.com/gpg -o "$key"
  [[ -s "$key" ]] || { echo "Downloaded signing key from https://apt.releases.hashicorp.com/gpg is empty"; exit 1; }
  sudo install -dm 755 /usr/share/keyrings
  sudo gpg --batch --yes --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg "$key"
  rm -f "$key"
  echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list >/dev/null
  # Drop the source again if apt can't use it, so later installers can still run apt update.
  if ! sudo apt -y update; then
    sudo rm -f /etc/apt/sources.list.d/hashicorp.list
    exit 1
  fi
  sudo apt install -y vagrant
  vagrant plugin install vagrant-docker-compose
fi
