### -- Manifest
### provides: common/node
### depends_on: [common/mise]
### distro: [ubuntu]
### -- End

echo "Installing Node.js via mise"
mise use --global node@lts
