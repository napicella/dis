### -- Manifest
### provides: common/autojump
### depends_on: []
### distro: [amazon_linux]
### -- End
source $DIS_BINDING

if command -v autojump &> /dev/null
then
    echo "autojump is installed"
    exit 0
fi

# Install autojump: https://www.linode.com/docs/guides/faster-file-navigation-with-autojump
git clone https://github.com/wting/autojump.git /tmp/autojump
cd /tmp/autojump
SHELL=/bin/bash ./install.py
cd /tmp && rm -rf /tmp/autojump

# Manual activation is required. 
# The following adds the activation in the bash init which is included in bashrc
bashrc_init_add "Autojump amzn linux" '[[ -s ~/.autojump/etc/profile.d/autojump.sh ]] && source ~/.autojump/etc/profile.d/autojump.sh'