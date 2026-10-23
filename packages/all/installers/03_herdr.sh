### -- Manifest
### provides: common/herdr
### depends_on: [common/os-libs]
### distro: [all]
### -- End

# https://herdr.dev - installed by its own installer to $HERDR_INSTALL_DIR (which its
# install.sh reads), and kept current with `herdr update`.
export HERDR_INSTALL_DIR=${HERDR_INSTALL_DIR:-$HOME/.local/bin}
if [[ -n "${DIS_INSTALL:-}" ]]; then
  mkdir -p ~/.config/herdr
  curl -fsSL https://herdr.dev/install.sh | sh
fi

# herdr's own Claude hook (agent state in the sidebar). Idempotent; it goes first so the
# title hook below joins its SessionStart block.
"$HERDR_INSTALL_DIR/herdr" integration install claude
mkdir -p ~/.claude/hooks && cp "$DIS_CONFIG_FOLDER/herdr/herdr-claude-title.sh" ~/.claude/hooks/
bash "$DIS_CONFIG_FOLDER/herdr/install-claude-title-hook.sh"
mkdir -p ~/.claude/skills && cp -r "$DIS_CONFIG_FOLDER/herdr/spin-chat-herdr" ~/.claude/skills/
# Official herdr skill, bundled in the binary so it matches the installed version; re-run
# `dis config herdr` after `herdr update` to refresh it.
mkdir -p ~/.claude/skills/herdr && "$HERDR_INSTALL_DIR/herdr" --skill > ~/.claude/skills/herdr/SKILL.md
mkdir -p ~/.config/herdr && cp "$DIS_CONFIG_FOLDER/herdr/herdr-keys.sh" "$DIS_CONFIG_FOLDER/herdr/herdr-shell-tab.sh" ~/.config/herdr/

# Config: always re-deploy the herdr config file, but keep the theme another tool (e.g. a
# theme engine) set in the deployed copy; config.toml.tmpl keeps theme.name and, when
# present, the [theme.custom] colors.
mkdir -p ~/.config/herdr
dis tools render-config "$DIS_CONFIG_FOLDER/herdr/config.toml.tmpl" ~/.config/herdr/config.toml
