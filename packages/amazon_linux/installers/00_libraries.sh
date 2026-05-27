### -- Manifest
### provides: common/os-libs
### depends_on: []
### distro: [amazon_linux]
### -- End

# Bootstrap: install sudo if running as root (e.g. in a fresh container).
if [[ $(whoami) == 'root' ]]; then
    yum update -y
    yum install -y sudo
fi

# Install the required packages
sudo yum groupinstall -y "Development Tools"
sudo yum install -y jq
sudo yum install -y \
    autoconf bison clang \
    openssl-devel readline-devel zlib-devel libyaml-devel readline-devel ncurses-devel libffi-devel gdbm-devel jemalloc-devel \
    socat sqlite sqlite-devel strace \
    tree
