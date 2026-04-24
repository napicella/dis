### -- Manifest
### provides: common/brew
### depends_on: []
### distro: [amazon_linux]
### -- End
source $DIS_BINDING

 export NONINTERACTIVE=1
 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"


# add the brew path last, favoring other tools first
bashrc_path_add 'Brew' 'export PATH=/home/linuxbrew/.linuxbrew/bin:$PATH'
# Adding brew to path so other installers can rely on it
export PATH=$PATH:/home/linuxbrew/.linuxbrew/bin

