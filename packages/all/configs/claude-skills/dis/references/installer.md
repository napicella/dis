# Writing a dis installer

Authoritative details: `docs/user-guide.md` in the dis repo (Manifest format,
Environment variables, RC helper tools) and `dis tools <command> --help`.

## Template

```bash
### -- Manifest
### provides: tools/foo
### depends_on: [common/os-libs]
### distro: [all]
### -- End

# Install-only steps: packages, downloads, builds. 'dis config' skips them.
if [[ -n "${DIS_INSTALL:-}" ]]; then
  case "$DIS_DISTRO" in
    amazon_linux) sudo yum install -y foo ;;
    ubuntu)       sudo DEBIAN_FRONTEND=noninteractive apt install -y foo ;;
    *)            echo "tools/foo: unsupported distro '$DIS_DISTRO'" >&2; exit 1 ;;
  esac
fi

# Config steps: run by both 'dis install' and 'dis config', so keep them idempotent.
mkdir -p ~/.config/foo
cp "$DIS_CONFIG_FOLDER/foo/config.toml" ~/.config/foo/config.toml

dis tools add-rc-aliases \
  --name 'foo aliases' \
  --content "alias f='foo --fast'"
```

## Manifest

| Field | Notes |
|---|---|
| `provides` | Unique `group/name`. Reuse the name only for OS variants of the same package. |
| `distro` | `[all]`, `[ubuntu]`, `[amazon_linux]`. One file per OS when the steps differ a lot, else one `[all]` file with a `case "$DIS_DISTRO"`. |
| `depends_on` | Packages that must run first, e.g. `common/go` before `go install`. |
| `requires_env` | Distro `parameters` (`FOO`, `FOO_*`) or another package's exports (`pkg:VAR`). |
| `exports_env` | Names written as `KEY=value` lines to `$DIS_EXPORTS_FILE`. |

## Install-only vs config

- **Inside `if [[ -n "${DIS_INSTALL:-}" ]]`:** only what is slow or needs the
  network or root: apt/yum, `go install` / `cargo install` of third-party
  tools, downloads.
- **Outside:** everything else — copying configs, `dis tools add-rc-*`,
  building tools from local source (so `dis config` picks up local changes).
- rc sections are config: `dis config` rewrites them and records their owner.
  Put `dis tools add-rc-*` before the guard when it can `exit` early (e.g.
  "already installed").
- Every step outside the guard must be safe to run again: `dis config` re-runs
  it on every call. `dis tools` rc helpers already are (they upsert).
- The script runs under `bash -e`: a failing command aborts the install.

## One-time migrations

Cleanup that only exists because of an older version of a package (a renamed
package's rc lock, a stale binary or unit file, a removed rc section) goes in a
config step marked with the date it was added:

```bash
# MIGRATION(2026-10-01): one-time cleanup; drop once every host has run dis config since then.
# <why the cleanup is needed>
if <old state is present>; then
  <cleanup>
fi
```

- Check for the old state first, so the block is a no-op once done.
- `dis search installers 'MIGRATION\('` lists the blocks still waiting to be removed.

## Environment

| Variable | Value |
|---|---|
| `DIS_PACKAGE` | This package's name; rc sections it writes are owned by it |
| `DIS_PKG_ROOT` | Package root from `dis.ws.yml`, or the source directory |
| `DIS_CONFIG_FOLDER` | Configs directory from `dis.ws.yml`; empty when none is declared |
| `DIS_INSTALLER` | Absolute path of this script (`$(dirname "$DIS_INSTALLER")` for files next to it) |
| `DIS_DISTRO` | The distro file's `os` |
| `DIS_INSTALL` | `1` under `dis install`, unset under `dis config` |
| `DIS_EXPORTS_FILE` | Where to write exports |

Reference config files as `$DIS_CONFIG_FOLDER/...`: that is how
`dis search configs .` finds them.

## RC helpers

Each upserts a named section in a file under `~/rc/configs-generated/`, which
`~/.bashrc` sources in this order. The files are rendered from
`~/.local/share/dis/sections.yaml`: never edit them, change the installer.

| Command | File | Use for |
|---|---|---|
| `dis tools add-rc-path` | `bash_paths` | `--path DIR` for `PATH` entries (prepended, skipped if already in PATH, so nested shells don't repeat them); `--content` for other `export`s. Also sourced before each installer, so later installers see them. |
| `dis tools add-rc-aliases` | `bash_aliases` | Aliases and functions. `--owner PKG` locks the section: only PKG can overwrite or remove it. |
| `dis tools add-rc-init` | `bash_init` | Code for interactive shells only (prompt hooks, completions). |
| `dis tools rm-rc-section --file FILE` | any of the three | Remove a section a package no longer provides. |
| `dis tools add-home-rc` | `~/.bashrc` | Wiring a top-level rc file; rarely needed. |

Section names must be unique per file; reuse the same name to update a section.
