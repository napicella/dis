### -- Manifest
### provides: common/java
### depends_on: [common/mise]
### distro: [all]
### -- End

# Java through mise.
if [[ -n "${DIS_INSTALL:-}" ]]; then
  # 22 is the default
  mise use --global java@corretto-22
fi
