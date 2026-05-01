### -- Manifest
### provides: common/node
### depends_on: [common/mise]
### distro: [all]
### -- End

source $DIS_BINDING

echo "Installing Node.js via mise"
mise use --global node@lts
