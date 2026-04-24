### -- Manifest
### provides: common/languages
### depends_on: [common/mise]
### distro: [ubuntu]
### -- End

source $DIS_BINDING

echo "Installing languages"

mise use --global node@lts
mise use --global python@latest
# mise use --global ruby@3.3

# In reality managing golang versions with mise is not necessary most of the times. Infact, golang is
# going to download the runtime if the go.mod points to a golang version that is not available in the system.
# The toolchain would be installed in "$GOPATH"/pkg/mod, eg:
# ls "$GOPATH"/pkg/mod/golang.org 
# toolchain@v0.0.1-go1.22.4.linux-amd64
#
# We make GOBIN available in the PATH.
mise use --global golang@latest
bashrc_path_add "GOBIN" 'export GOBIN=$(go env GOBIN)
export GOPATH=$(go env GOPATH)
export GOPROXY=direct
export PATH=$GOBIN:$PATH
'