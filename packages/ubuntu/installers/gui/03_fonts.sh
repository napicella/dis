### -- Manifest
### provides: common/fonts
### depends_on: [common/gum]
### distro: [ubuntu]
### -- End
source $DIS_BINDING

# Make fonts.sh available as a standalone tool in the user's shell.
# Use DOTFILES_FOLDER (available after bashrc install) for the persistent PATH entry.
# Use DIS_PKG_ROOT for the current installer session.
bashrc_path_add "Ubuntu tools" 'export PATH="$DOTFILES_FOLDER/dis/packages/ubuntu/bin:$PATH"'
# Also add it to PATH for the remainder of this installer session.
export PATH="$DIS_PKG_ROOT/bin:$PATH"

fonts "Cascadia Mono"
