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

mkdir -p ~/.claude/hooks && cp "$DIS_CONFIG_FOLDER/herdr/herdr-claude-title.sh" ~/.claude/hooks/
bash "$DIS_CONFIG_FOLDER/herdr/install-claude-title-hook.sh"
mkdir -p ~/.claude/skills && cp -r "$DIS_CONFIG_FOLDER/herdr/spin-chat-herdr" ~/.claude/skills/
mkdir -p ~/.config/herdr && cp "$DIS_CONFIG_FOLDER/herdr/herdr-keys.sh" ~/.config/herdr/

# Config: always re-deploy the herdr config file, but keep the theme another tool (e.g. a
# theme engine) set in the deployed copy; config.toml.tmpl keeps theme.name and, when
# present, the [theme.custom] colors.
mkdir -p ~/.config/herdr
dis tools render-config "$DIS_CONFIG_FOLDER/herdr/config.toml.tmpl" ~/.config/herdr/config.toml
