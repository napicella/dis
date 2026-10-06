### -- Manifest
### provides: common/herdr
### depends_on: []
### distro: [all]
### -- End

# https://herdr.dev - installed to ~/.local/bin by its own installer, and kept current
# with `herdr update`.
if [[ -n "${DIS_INSTALL:-}" ]]; then
  mkdir -p ~/.config/herdr
  curl -fsSL https://herdr.dev/install.sh | sh
fi

# MIGRATION(2026-10-01): one-time cleanup; drop once every host has run dis config since then.
# herdr used to be in the global mise config as well. mise's copy came first on PATH,
# and `herdr update` replaced its binary behind mise's back. Drop it once. unuse only
# edits the config, so also uninstall mise's copy, which removes its shim too.
if command -v mise &> /dev/null && mise config get --global tools.herdr &> /dev/null; then
  mise unuse --global herdr
  mise uninstall --all herdr || true
fi

mkdir -p ~/.claude/hooks && cp "$DIS_CONFIG_FOLDER/herdr/herdr-claude-title.sh" ~/.claude/hooks/
bash "$DIS_CONFIG_FOLDER/herdr/install-claude-title-hook.sh"
mkdir -p ~/.claude/skills && cp -r "$DIS_CONFIG_FOLDER/herdr/spin-chat-herdr" ~/.claude/skills/
mkdir -p ~/.config/herdr && cp "$DIS_CONFIG_FOLDER/herdr/herdr-keys.sh" ~/.config/herdr/

# Config: always re-deploy the herdr config file, but keep the theme wal-picker set in
# the deployed copy; config.toml marks theme.name and the [theme.custom] table with keep.
mkdir -p ~/.config/herdr
dis tools render-config "$DIS_CONFIG_FOLDER/herdr/config.toml.tmpl" ~/.config/herdr/config.toml
