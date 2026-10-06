### -- Manifest
### provides: common/os-libs
### depends_on: []
### distro: [amazon_linux]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  # Bootstrap: install sudo if running as root (e.g. in a fresh container).
  if [[ $(whoami) == 'root' ]]; then
      yum update -y
      yum install -y sudo
  fi

  # Install the required packages
  sudo yum groupinstall -y "Development Tools"
  sudo yum install -y \
      wget curl zip unzip tar gzip jq xz which gnupg2 gettext hostname \
      autoconf bison clang \
      openssl-devel zlib-devel libyaml-devel readline-devel ncurses-devel libffi-devel gdbm-devel jemalloc-devel \
      socat sqlite sqlite-devel strace \
      tree
fi

# Amazon Linux sets the locale only in login shells (/etc/profile.d/lang.sh), so
# shells started otherwise (ssh host cmd, VS Code remote, a multiplexer server)
# fall back to the ASCII C locale. Fill in LANG when it's missing.
dis tools add-rc-env \
  --name 'Locale' \
  --content 'export LANG="${LANG:-en_US.UTF-8}"'
