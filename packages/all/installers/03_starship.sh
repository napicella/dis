### -- Manifest
### provides: common/starship
### depends_on: [common/yq]
### distro: [all]
### -- End

# Wire starship into the shell RC.
dis tools add-rc-init \
  --name 'Starship' \
  --content $'if [[ $- == *i* ]] && [[ ${TERM:-} != "dumb" ]] && command -v starship &> /dev/null; then
  eval "$(starship init bash)"
fi'

if [[ -n "${DIS_INSTALL:-}" ]]; then
  # Use this method only if the Ubuntu version is earlier than 25.04.
  # Ubuntu 25.04 and later provide the package in the Ubuntu repository.
  # https://starship.rs/guide/
  VER=$(. /etc/os-release && echo $VERSION_ID | sed 's/\.//')
  if (( $VER >= 2504 )); then
    sudo apt -y install starship
  else
    # not ubuntu or ubuntu earlier than 25.04
    # Download the script first: piped into sh, a failed download would run an
    # empty (or error page) script and still count as a success.
    script=$(mktemp)
    curl -fsSL https://starship.rs/install.sh -o "$script"
    sh "$script" --yes
    rm -f "$script"
  fi
fi

# Config: always re-deploy the starship config file, but keep the colors wal-picker
# set in the deployed copy (palette and [palettes.custom]); a plain copy resets them.
_cfg=~/.config/starship.toml
_palette= _main= _secondary=
if [[ -f "$_cfg" ]] && command -v yq &> /dev/null; then
  _palette=$(yq -p toml -oy '.palette // ""' "$_cfg")
  _main=$(yq -p toml -oy '.palettes.custom.main_color // ""' "$_cfg")
  _secondary=$(yq -p toml -oy '.palettes.custom.secondary_color // ""' "$_cfg")
fi
mkdir -p ~/.config/
cp "$DIS_CONFIG_FOLDER/starship/starship.toml" "$_cfg"
# starship config edits the file in place and keeps its formatting.
if command -v starship &> /dev/null; then
  if [[ -n "$_palette" ]]; then starship config palette "$_palette"; fi
  if [[ -n "$_main" ]]; then starship config palettes.custom.main_color "$_main"; fi
  if [[ -n "$_secondary" ]]; then starship config palettes.custom.secondary_color "$_secondary"; fi
fi
