### -- Manifest
### provides: common/go
### depends_on: [common/mise]
### distro: [all]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  echo "Installing Go via mise"
  mise use --global golang@latest
fi

# By default mise sets GOBIN to ~/.local/share/mise/installs/go/<version>/bin, so
# `go install`ed tools are tied to one Go version. After a Go upgrade, their shims
# fail with "No version is set for shim". Turning this off keeps Go's default
# $GOPATH/bin, including for callers that only go through shims (cron, systemd, IDEs).
mise settings set go.set_gobin false

# MIGRATION(2026-10-05): one-time cleanup; drop once every host has run dis config since then.
# Tools `go install`ed while set_gobin was on are still in the versioned bin dirs, and
# mise keeps a shim for each. Once the active Go lacks the tool, the shim fails with
# "No version is set for shim" in any process that inherited a mise-activated env (an
# IDE that resolved the shell env): mise drops the PATH dirs it added, ~/go/bin among
# them, from the shim's fallback search. The mise VS Code extension also points
# go.alternateTools at any gopls/dlv shim that exists. Move the leftovers to ~/go/bin
# (or drop them if it has its own copy) and reshim, so the shims go away.
go_bin="$HOME/go/bin"
mkdir -p "$go_bin"
for f in "${MISE_DATA_DIR:-$HOME/.local/share/mise}"/installs/go/*/bin/*; do
  name=$(basename "$f")
  [[ -f "$f" && ! -L "$f" && "$name" != go && "$name" != gofmt ]] || continue
  if [[ -e "$go_bin/$name" ]]; then
    rm -f "$f"
  else
    mv "$f" "$go_bin/"
  fi
done
mise reshim

# PATH priority, read only by mise: activated shells and shim-launched processes.
# mise activate moves the shims right behind the active versions' dirs, ahead of
# anything in bash_paths. Adding GOBIN through mise puts it first instead, so
# go-installed tools don't pass through a stale shim.
mise config set --global --append --type list env._.path '~/go/bin'

# Saved in Go's own env file, which every go call reads (shells, IDEs, cron, shims),
# unlike an rc export.
go env -w GOPROXY=direct

# GOPATH is left unset: Go's default is ~/go. GOBIN is exported to override a stale
# value inherited from the login session (e.g. a versioned dir from the old setup),
# and it's set statically: `$(go env GOBIN)` ran go through the shim at every shell
# start. The PATH line is for dis installers: the wrapper sources bash_env and
# bash_paths without mise activate, so `_.path` above doesn't reach them.
dis tools add-rc-env \
  --name 'GOBIN env' \
  --content 'export GOBIN="$HOME/go/bin"'
dis tools add-rc-path \
  --name 'GOBIN' \
  --path '$HOME/go/bin'
