### -- Manifest
### provides: common/languages
### depends_on: [common/mise,common/brew]
### distro: [amazon_linux]
### -- End
source $DIS_BINDING
echo "Installing languages"

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
export PATH=$GOBIN:$PATH
'

# For amazon_linux we assume the old AL2 distro which ships with an old version of glibc.
# That prevent installing node and likely other languages from source like mise does. So
# we use both mise and brew to set up languages.
#
# Update (August 13, 2025): Amazon/Builder Tools seems to have fixed the issue. So now,
# it should be possible to install languages from source.

brew install node
brew install python

 
