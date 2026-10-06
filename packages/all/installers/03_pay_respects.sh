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
  # Build with the physical path of CARGO_HOME. pay-respects' templates (askama)
  # are found through a relative path with '..', which leaves the crate when the
  # home dir is a symlink (Cloud Desktops: /home/x -> /local/home/x), and the
  # build fails with "couldn't read .../templates/init.bash".
  _cargo_home="${CARGO_HOME:-$HOME/.cargo}"
  mkdir -p "$_cargo_home"
  CARGO_HOME="$(cd "$_cargo_home" && pwd -P)" cargo install --locked pay-respects
fi

# --nocnf: leave the shell's command-not-found handler alone (Ubuntu's suggests
# packages), as thefuck did.
# Interactive shells only: its setup binds a key, which warns "line editing not
# enabled" in shells that read the rc without being interactive, like the
# 'bash -c' pay-respects runs the fixed command in, over SSH on Debian/Ubuntu.
dis tools add-rc-init \
  --name 'pay-respects' \
  --content 'if [[ $- == *i* ]] && command -v pay-respects &> /dev/null; then
  eval "$(pay-respects bash --alias fuck --nocnf)"
  eval "$(pay-respects bash --alias please --nocnf)"
fi'
