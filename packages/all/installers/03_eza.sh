### -- Manifest
### provides: common/eza
### depends_on: [tools/cargo]
### distro: [all]
### -- End

# https://github.com/eza-community/eza - modern alternative to file listing

if [[ -n "${DIS_INSTALL:-}" ]]; then
  echo "Installing eza via cargo..."
  cargo install --locked eza
fi

dis tools add-rc-aliases \
  --name 'ls aliases' \
  --owner 'common/eza' \
  --content "$(cat <<'EOF'
if command -v eza &>/dev/null; then
  alias ls="eza --icons=auto"
  alias ll="eza -la --icons=auto"
fi
EOF
)"
