### -- Manifest
### provides: common/smartcd
### depends_on: [common/os-libs]
### distro: [all]
### -- End

## Install smartcd (https://github.com/cxreg/smartcd)
# 
# You can create or modify the smartcd config via the "smartcd config command" (or create/edit it manually). When using
# the config command the config is stored in  $HOME/.smartcd_config. For the tool to work you need to source
# the config (i.e. $HOME/.smartcd_config) config in your bashrc. 
# More docs here: https://github.com/cxreg/smartcd/blob/master/lib/core/smartcd
#
# To add a new behavior when entering/leaving a dir, you can use `smartcd edit enter` to start an interactive editor or
# do it programmatically, for example: echo 'autostash alias svc="cd __PATH__/to/somewhere"' | smartcd edit enter
#
# ## Installation details
# "make install" the tool  in the $HOME/.smartcd folder. The folder contains the source bash scripts and the files used
# by the user to configure the action to perform when entering/leaving directory (files created with smartcd edit
# enter). In particular the latter are in the  $HOME/.smartcd/scripts folder.
# 
# The installation process thus create the following:
# - $HOME/.smartcd/* (created by make install)
# - $HOME/.smartcd_config (config file, created by dis)
#
# To remove smartcd from your system run the following command:
# rm -rf $HOME/.smartcd $HOME/.smartcd_config && source $HOME/.bashrc

source $DIS_BINDING

if command -v smartcd &> /dev/null
then
    echo "smartcd is installed"
    exit 0
fi

git clone https://github.com/cxreg/smartcd.git /tmp/smartcd
cd /tmp/smartcd && make install && cd -
# remove repo after the installation
rm -rf /tmp/smartcd

# do not use the interactive command to create the config (source load_smartcd && smartcd config). Use the one defined
# in the dis installer.
cp $DIS_CONFIG_FOLDER/.smartcd_config $HOME/
# Manual activation is required. 
# The following adds the activation in the bash init which is included in bashrc
bashrc_init_add "smartcd (https://github.com/cxreg/smartcd)" '[ -r "$HOME/.smartcd_config" ] && ( [ -n $BASH_VERSION ] || [ -n $ZSH_VERSION ] ) && source ~/.smartcd_config'