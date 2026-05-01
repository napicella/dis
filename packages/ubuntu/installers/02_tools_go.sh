### -- Manifest
### provides: common/go
### depends_on: [common/mise]
### distro: [all]
### -- End

source $DIS_BINDING

echo "Installing Go via mise"
mise use --global golang@latest

bashrc_path_add "GOBIN" 'export GOBIN=$(go env GOBIN)
export GOPATH=$(go env GOPATH)
export GOPROXY=direct
export PATH=$GOBIN:$PATH
'
