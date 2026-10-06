### -- Manifest
### provides: common/java
### depends_on: [common/mise]
### distro: [all]
### -- End

# Java through mise, which replaced sdkman (common/sdkman before). mise's java
# plugin installs the Amazon Corretto builds and, in shells with mise activate,
# sets JAVA_HOME, as sdkman did. Commands that only go through the shims (e.g.
# 'ssh host cmd', cron) get java but no JAVA_HOME. Other versions are one
# command away: 'mise use java@corretto-8' in a project dir.
if [[ -n "${DIS_INSTALL:-}" ]]; then
  # 22 is the default
  mise use --global java@corretto-22

  # MIGRATION(2026-10-06): one-time cleanup; drop once every host has run dis install since then.
  # Remove sdkman and the JDKs it installed, now that mise provides Java.
  if [[ -d "$HOME/.sdkman" ]]; then
    rm -rf "$HOME/.sdkman"
  fi
fi

# MIGRATION(2026-10-06): one-time cleanup; drop once every host has run dis config since then.
# common/sdkman added this section to load sdkman in interactive shells.
dis tools rm-rc-section --file bash_init --name 'Sdkman'
