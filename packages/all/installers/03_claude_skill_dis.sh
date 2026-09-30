### -- Manifest
### provides: common/claude-skill-dis
### depends_on: []
### distro: [all]
### -- End

# Install the Claude Code skill that teaches Claude how to work with dis
# (search, list sources, write installers). It is a user-level skill, so it is
# available in every session, whatever the working directory.
if [[ ! -d ~/.claude ]]; then
  echo "common/claude-skill-dis: ~/.claude not found (Claude Code not set up), skipping"
  exit 0
fi

# Replace the whole skill so files removed from the repo do not linger.
mkdir -p ~/.claude/skills
rm -rf ~/.claude/skills/dis
cp -r "$DIS_CONFIG_FOLDER/claude-skills/dis" ~/.claude/skills/dis
