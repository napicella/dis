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
    --path "$DIS_PKG_ROOT/bin"
  # Also add it to PATH for the remainder of this installer session.
  export PATH="$DIS_PKG_ROOT/bin:$PATH"

  fonts "Cascadia Mono"

  # Nerd Fonts only patch in icon sets, not general Unicode symbols such as
  # U+23F5 (used by the Claude Code status line). Noto Sans Symbols 2 covers
  # these and fontconfig picks it up as a fallback automatically.
  if ! fc-list ':charset=23f5' family | grep -q .; then
    sudo apt-get install -y fonts-noto-core
    fc-cache -f
  fi
fi
