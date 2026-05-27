### -- Manifest
### provides: common/brew
### depends_on: []
### distro: [amazon_linux]
### -- End
export NONINTERACTIVE=1
 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"


# add the brew path last, favoring other tools first
dis tools add-rc-path --name 'Brew' --content 'export PATH=/home/linuxbrew/.linuxbrew/bin:$PATH'
# Adding brew to path so other installers can rely on it
export PATH=$PATH:/home/linuxbrew/.linuxbrew/bin

