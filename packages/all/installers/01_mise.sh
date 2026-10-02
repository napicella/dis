### -- Manifest
### provides: common/mise
### depends_on: [common/os-libs]
### distro: [all]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  if command -v mise &> /dev/null; then
    echo "mise is installed"
  else
    # Install mise for managing multiple versions of languages. See https://mise.jdx.dev/
    case "$DIS_DISTRO" in
      ubuntu)
        # Download the key to a file first: in a pipeline a failed download goes unnoticed and
        # leaves an empty keyring behind a source that breaks every later 'apt update'.
        key=$(mktemp)
        curl -fsSL https://mise.jdx.dev/gpg-key.pub -o "$key"
        [[ -s "$key" ]] || { echo "Downloaded signing key from https://mise.jdx.dev/gpg-key.pub is empty"; exit 1; }
        sudo install -dm 755 /etc/apt/keyrings
        sudo gpg --batch --yes --dearmor -o /etc/apt/keyrings/mise-archive-keyring.gpg "$key"
        rm -f "$key"
        echo "deb [signed-by=/etc/apt/keyrings/mise-archive-keyring.gpg arch=amd64] https://mise.jdx.dev/deb stable main" | sudo tee /etc/apt/sources.list.d/mise.list >/dev/null
        # Drop the source again if apt can't use it, so later installers can still run apt update.
        if ! sudo apt update -y; then
          sudo rm -f /etc/apt/sources.list.d/mise.list
          exit 1
        fi
        sudo apt install -y mise
        ;;
      amazon_linux)
        # Install it under /usr/bin like package managers do
        curl https://mise.run -o /tmp/install-mise.sh && chmod +x /tmp/install-mise.sh
        sudo MISE_INSTALL_PATH=/usr/bin/mise /tmp/install-mise.sh
        ;;
      *)
        echo "common/mise: unsupported distro '$DIS_DISTRO'" >&2
        exit 1
        ;;
    esac
  fi
fi

# Both install methods put the mise cli under /usr/bin, so no need to add that to the path.
# mise puts the active version of each tool on PATH in two ways: activation and shims.
# We use both, as upstream recommends (https://mise.jdx.dev/dev-tools/shims.html).
#
# The active version of a tool comes from config files: mise reads mise.toml from the
# current dir up to /, then ~/.config/mise/config.toml, and the closest file that
# names the tool wins. `mise use -g tool@ver` installs the version and writes it to the
# global file. Without -g it writes ./mise.toml, a per-project pin. `mise install` only
# downloads, so a version that no config file names is installed but inactive.
#
# Activation: for interactive shells. A prompt/cd hook puts the real bin dirs of the
# active versions on PATH, ahead of the shims below. It also applies mise [env] vars and
# hooks, and `which` shows the real binary. bash_init is sourced after bash_paths, so
# these dirs win over the shims line there.
dis tools add-rc-init \
  --name 'Mise activate' \
  --content 'eval "$(mise activate bash)"'

# Shims: for everything that never runs the activation hook. One symlink to mise per
# tool binary, covering every installed version. mise sees which name it was called as,
# resolves the active version for the caller's dir as above, and execs that version's
# binary, so no shell hook is needed. The dis wrapper sources bash_paths (not
# bash_init) before each installer, so installers go through the shims. Cron, systemd
# and IDEs don't read the rc, so they need this dir in their own PATH (or
# `mise exec --`). If no active version has the binary (e.g. a tool left under an older
# Go), the shim runs the next match on PATH, else fails with "No version is set for
# shim". Shims exist only for binaries inside mise's install dirs, and mise updates
# them only when it installs or uninstalls. A binary put there behind its back needs
# `mise reshim`, e.g. `pip install` into mise's python, or `go install` with mise's
# default GOBIN. Tools installed outside mise's dirs never get a shim. They're reached
# through PATH instead and never need a reshim, like go tools once common/go sets
# GOBIN=~/go/bin.
dis tools add-rc-path --name 'Mise path' --path '$HOME/.local/share/mise/shims'

# Resulting interactive PATH, highest priority first:
#   ~/.local/share/mise/installs/<tool>/<ver>/bin      mise activate: real dirs of the active versions,
#                                                      rewritten on every prompt and cd; never hand-edit
#   
#   ~/.local/share/mise/shims                          backstop only
#
#   ~/.local/bin, script dirs, linuxbrew               your scripts, other package managers 
#                               
#   /usr/local/sbin ... /bin, /snap/bin                system
#
# Non-interactive processes (cron, systemd, an IDE without the mise extension) get
# no activate block, so the shims stand in for it:
#   ~/go/bin:~/.local/share/mise/shims:~/.local/bin:~/.toolbox/bin:/usr/local/bin:/usr/bin:/bin