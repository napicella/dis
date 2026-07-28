### -- Manifest
### provides: tools/cargo
### depends_on: [common/mise]
### distro: [all]
### -- End

echo "Installing Rust/Cargo via mise"
mise use --global rust@latest

dis tools add-rc-path \
  --name 'Cargo' \
  --content 'export PATH="$HOME/.cargo/bin:$PATH"'
