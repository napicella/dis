### -- Manifest
### provides: common/mkcert
### depends_on: [common/languages]
### distro: [all]
### -- End
source $DIS_BINDING
# Install https://github.com/FiloSottile/mkcert
go install filippo.io/mkcert@latest