### -- Manifest
### provides: common/go
### depends_on: [common/mise]
### distro: [ubuntu]
### -- End

echo "Installing Go via mise"
mise use --global golang@latest

dis tools add-rc-path \
  --name 'GOBIN' \
  --content 'export GOBIN=$(go env GOBIN)
export GOPATH=$(go env GOPATH)
export GOPROXY=direct
export PATH=$GOBIN:$PATH'
