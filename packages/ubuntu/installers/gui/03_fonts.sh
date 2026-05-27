### -- Manifest
### provides: gui/fonts
### depends_on: [common/gum]
### distro: [ubuntu]
### -- End

# Make fonts.sh available as a standalone tool in the user's shell.
# DIS_PKG_ROOT is expanded now (at install time) so the absolute path is baked
# into ~/.bashrc for future shell sessions.
bashrc_path_add "Ubuntu tools" "export PATH=\"$DIS_PKG_ROOT/bin:\$PATH\""
# Also add it to PATH for the remainder of this installer session.
export PATH="$DIS_PKG_ROOT/bin:$PATH"

fonts "Cascadia Mono"
