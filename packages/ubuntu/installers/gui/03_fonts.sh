### -- Manifest
### provides: gui/fonts
### depends_on: [common/gum]
### distro: [ubuntu]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  # Make fonts.sh available as a standalone tool in the user's shell.
  # DIS_PKG_ROOT is expanded now (at install time) so the absolute path is baked
  # into ~/.bashrc for future shell sessions.
  dis tools add-rc-path \
    --name 'Ubuntu tools' \
    --content "export PATH=\"$DIS_PKG_ROOT/bin:\$PATH\""
  # Also add it to PATH for the remainder of this installer session.
  export PATH="$DIS_PKG_ROOT/bin:$PATH"

  fonts "Cascadia Mono"
fi
