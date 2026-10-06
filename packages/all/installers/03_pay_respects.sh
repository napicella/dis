### -- Manifest
### provides: common/pay-respects
### depends_on: [tools/cargo]
### distro: [all]
### -- End

# https://github.com/iffse/pay-respects - corrects the previous command, a thefuck
# replacement written in Rust. thefuck (common/thefuck before) is unmaintained and
# breaks on Python 3.12+ (it imports the removed 'imp' module), e.g. Ubuntu 24.04.

if [[ -n "${DIS_INSTALL:-}" ]]; then
  echo "Installing pay-respects via cargo..."
  cargo install --locked pay-respects

  # MIGRATION(2026-10-06): one-time cleanup; drop once every host has run dis install since then.
  # Remove thefuck, which common/thefuck installed.
  case "$DIS_DISTRO" in
    ubuntu)
      if dpkg -s thefuck &> /dev/null; then
        sudo DEBIAN_FRONTEND=noninteractive apt-get remove -y thefuck
      fi
      ;;
    amazon_linux)
      if command -v brew &> /dev/null && brew list thefuck &> /dev/null; then
        brew uninstall thefuck
      fi
      ;;
  esac
fi

# MIGRATION(2026-10-06): one-time cleanup; drop once every host has run dis config since then.
# Drop the section common/thefuck added: a broken thefuck prints a traceback on
# every shell start.
dis tools rm-rc-section --file bash_init --name 'TheFuck'

# --nocnf: leave the shell's command-not-found handler alone (Ubuntu's suggests
# packages), as thefuck did.
dis tools add-rc-init \
  --name 'pay-respects' \
  --content 'if command -v pay-respects &> /dev/null; then
  eval "$(pay-respects bash --alias fuck --nocnf)"
  eval "$(pay-respects bash --alias please --nocnf)"
fi'
