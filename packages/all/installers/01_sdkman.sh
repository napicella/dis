### -- Manifest
### provides: common/sdkman
### depends_on: []
### distro: [all]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  # Install sdkman: https://sdkman.io/install
  if command -v sdk &> /dev/null; then
    echo "sdkman is installed"
    exit 0
  fi

  # Download the script first: piped into bash, a failed download would run an
  # empty (or error page) script and still count as a success.
  script=$(mktemp)
  curl -fsSL "https://get.sdkman.io" -o "$script"
  bash "$script"
  rm -f "$script"

  dis tools add-rc-init \
    --name 'Sdkman' \
    --content 'export SDKMAN_DIR="$HOME/.sdkman"
[[ -s "$HOME/.sdkman/bin/sdkman-init.sh" ]] && source "$HOME/.sdkman/bin/sdkman-init.sh"'
fi
