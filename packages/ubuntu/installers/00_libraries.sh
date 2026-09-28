### -- Manifest
### provides: common/os-libs
### depends_on: []
### distro: [ubuntu]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  # Bootstrap: install sudo if running as root (e.g. in a fresh container).
  if [[ $(whoami) == 'root' ]]; then
      DEBIAN_FRONTEND=noninteractive apt update -y
      DEBIAN_FRONTEND=noninteractive apt install -y sudo
  fi

  # Default the timezone on machines where it was never set, so tzdata doesn't prompt.
  if [[ ! -e /etc/localtime ]]; then
    echo 'Europe/Rome' | sudo tee /etc/timezone >/dev/null
    sudo ln -fs /usr/share/zoneinfo/Europe/Rome /etc/localtime
  fi

  # DEBIAN_FRONTEND is passed through sudo explicitly: sudo resets the environment.
  sudo DEBIAN_FRONTEND=noninteractive apt update -y
  sudo DEBIAN_FRONTEND=noninteractive apt install -y make wget curl zip unzip tar git tree gpg apt-utils gettext-base jq tzdata \
  	build-essential pkg-config autoconf bash-completion bison clang \
  	sqlite3 libsqlite3-0 \
  	xclip \
  	img2pdf

  	# libssl-dev libreadline-dev zlib1g-dev libyaml-dev libreadline-dev libncurses5-dev libffi-dev libgdbm-dev libjemalloc2 \
  	# libvips imagemagick libmagickwand-dev \
fi
