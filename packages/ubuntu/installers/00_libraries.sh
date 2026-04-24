### -- Manifest
### provides: common/os-libs
### depends_on: []
### distro: [ubuntu]
### -- End
source $DIS_BINDING

export DEBIAN_FRONTEND=noninteractive

# Bootstrap: install sudo if running as root (e.g. in a fresh container).
if [[ $(whoami) == 'root' ]]; then
    apt update -y
    apt install -y sudo
fi

sudo apt update -y
sudo apt install -y make wget curl zip unzip tar git tree gpg apt-utils gettext-base jq \
	build-essential pkg-config autoconf bash-completion bison clang \
	sqlite3 libsqlite3-0 \
	xclip \
	img2pdf

	# libssl-dev libreadline-dev zlib1g-dev libyaml-dev libreadline-dev libncurses5-dev libffi-dev libgdbm-dev libjemalloc2 \
	# libvips imagemagick libmagickwand-dev \
