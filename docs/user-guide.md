# dis — User Guide

## What is dis?

`dis` is a CLI for automating machine setup. You describe *what* to install in a distro YAML file and *how* to install it in shell scripts called installers. `dis` resolves dependencies between installers and runs them in the correct order, skipping anything already done.

---

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/install.sh | bash
```

To install to a custom location:

```bash
INSTALL_DIR=/usr/local/bin bash <(curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/install.sh)
```

This installs the `dis` binary to `~/.local/bin` (or `INSTALL_DIR`) and downloads the built-in common packages to `~/.local/share/dis/packages`.

To keep the built-in packages up to date:

```bash
dis sync
```

---

## Getting started

### 1. Scaffold a workspace

```bash
mkdir my-dotfiles && cd my-dotfiles
dis init
```

This creates:

```
dis.ws.yml
distro.yml
packages/hello/installers/hello.sh
```

### 2. Understand the distro file

`distro.yml` tells dis what to install:

```yaml
os: ubuntu

parameters:
  MY_PARAM: my-value

sources:
  - .                  # this directory (dis reads dis.ws.yml here)
  - ${common_sources}  # built-in dis packages

packages:
  - hello/greet
```

- **`os`** — the target operating system (`ubuntu`, `amazon_linux`, or `all`).
- **`parameters`** — static key/value pairs injected into installers that declare them in `requires_env`. These are *global* parameters available to every package. Supports `${home}` which expands to the user's home directory.
- **`sources`** — directories dis walks to discover installer scripts. If a `dis.ws.yml` is present in a source, dis uses it to scope which subdirectories to walk.
- **`packages`** — the list of package names to install, in the order you declare them. Transitive dependencies are resolved automatically.

### 3. Write an installer

Each installer is a `.sh` file with a manifest header block:

```bash
#!/usr/bin/env bash
### -- Manifest
### provides: hello/greet
### depends_on: []
### distro: [ubuntu]
### requires_env: [MY_PARAM]
### -- End

echo "Installing with MY_PARAM=${MY_PARAM}"
```

The manifest block tells dis everything it needs to know. No `source` line is needed — dis wraps every installer automatically (see [The wrapper](#the-wrapper) below).

### 4. Preview and install

```bash
# See what would run, in order, without executing anything
dis plan --distro distro.yml

# Run the installation
dis install --distro distro.yml
```

---

## Manifest format

The manifest is a comment block at the top of an installer script, between `### -- Manifest` and `### -- End`.

| Field | Required | Description |
|---|---|---|
| `provides` | yes | Unique fully-qualified name for this installer, e.g. `common/tmux` |
| `distro` | yes | List of OS names this installer applies to: `[ubuntu]`, `[amazon_linux]`, `[all]` |
| `depends_on` | no | List of `provides` names that must run before this installer |
| `requires_env` | no | Environment variables that dis injects before running this script |
| `exports_env` | no | Variables this script writes to `$DIS_EXPORTS_FILE` for downstream installers |

List fields may span multiple lines using continuation lines:

```bash
### requires_env: [DOCKER_MOUNT_FOLDER, WIREGUARD_KEY_PATH,
###                GID_RENDER, GID_ADM, UID_CONTAINER]
```

### requires_env syntax

- **Bare name** (`FOO`) — injected from the distro's `parameters` map.
- **Bare glob** (`FOO_*`) — all parameters matching that prefix are injected.
- **Qualified name** (`pkg:VAR`) — value exported by a prior installer (`pkg`).
- **Qualified glob** (`pkg:PREFIX*`) — all exports from `pkg` matching the prefix.

### Exporting values

Write `KEY=value` lines to `$DIS_EXPORTS_FILE`:

```bash
GID_DOCKER=$(getent group docker | cut -d: -f3)
echo "GID_DOCKER=${GID_DOCKER}" >> "$DIS_EXPORTS_FILE"
```

Exported values are stored in a persistent cache at `~/.local/share/dis/exports-cache.txt` and are available in future runs even when the exporting package is skipped as already-installed.

---

## Workspace file (dis.ws.yml)

When a source directory contains a `dis.ws.yml`, dis uses it to determine which subdirectories to walk and what `DIS_CONFIG_FOLDER` to set for each package.

```yaml
packages:
  - root: ./all          # walk this subdirectory for installers
    configs: ./all/configs  # optional: sets DIS_CONFIG_FOLDER for these installers
  - root: ./ubuntu
    configs: ./ubuntu/configs
```

Without `dis.ws.yml`, dis walks the entire source directory.

---

## Environment variables available in installers

The following variables are set by dis in the installer's environment:

| Variable | Description |
|---|---|
| `DIS_PKG_ROOT` | The `root` declared in `dis.ws.yml` for this package, or the source directory itself when no workspace file is present |
| `DIS_CONFIG_FOLDER` | Configs directory (set from `dis.ws.yml`; empty if not declared) |
| `DIS_INSTALLER` | Absolute path to the installer script |
| `DIS_DISTRO` | OS name from the distro YAML |
| `DIS_EXPORTS_FILE` | Temp file to write `KEY=value` exports for downstream installers |
| `DIS_INSTALL` | Set to `"1"` by `dis install`; absent when `dis config` runs. Use this to guard install-only steps (see [config command](#the-config-command)) |

### Concrete examples

**Example 1 — source without a workspace file** (e.g. a single-tool directory)

Given this distro source entry:
```
~/dotfiles/tools/env-manager
```
with no `dis.ws.yml` present, and an installer at:
```
~/dotfiles/tools/env-manager/env_manager_installer.sh
```
dis sets:
```
DIS_PKG_ROOT   = /home/nicola/dotfiles/tools/env-manager
DIS_INSTALLER  = /home/nicola/dotfiles/tools/env-manager/env_manager_installer.sh
DIS_CONFIG_FOLDER = (empty — not declared)
```

**Example 2 — source with a dis.ws.yml** (e.g. the built-in packages)

Given this `dis.ws.yml` in `~/.local/share/dis/packages`:
```yaml
packages:
  - root: ./all
    configs: ./all/configs
```
and an installer at:
```
~/.local/share/dis/packages/all/installers/00_bash_config.sh
```
dis sets:
```
DIS_PKG_ROOT      = /home/nicola/.local/share/dis/packages/all
DIS_INSTALLER     = /home/nicola/.local/share/dis/packages/all/installers/00_bash_config.sh
DIS_CONFIG_FOLDER = /home/nicola/.local/share/dis/packages/all/configs
```

**Example 3 — exporting and importing values between packages**

`producer.sh` exports a value:
```bash
### -- Manifest
### provides: myapp/docker
### exports_env: [GID_DOCKER]
### -- End

GID_DOCKER=$(getent group docker | cut -d: -f3)
echo "GID_DOCKER=${GID_DOCKER}" >> "$DIS_EXPORTS_FILE"
```

`consumer.sh` imports it using the qualified `pkg:VAR` syntax:
```bash
### -- Manifest
### provides: myapp/containers
### depends_on: [myapp/docker]
### requires_env: [myapp/docker:GID_DOCKER]
### -- End

echo "Docker GID is: ${GID_DOCKER}"
```

dis injects `GID_DOCKER` (the bare name, without the `myapp/docker:` prefix) into the consumer's environment.

---

## The wrapper

dis wraps every installer in a small shell script (`wrapper.sh`) before running it. The wrapper:

1. Creates `~/rc/configs-generated/` and ensures `bash_paths`, `bash_init`, and `bash_aliases` exist.
2. Sources `bash_paths` and `bash_aliases` so PATH additions from earlier installers propagate to the current one.
3. Prepends the `dis` binary directory to `PATH` so installers can call `dis tools ...` directly.
4. Runs the installer with `bash -e` (exit on error).

You do **not** need to add any `source` line to your installer scripts.

---

## RC helper tools

Installers that need to register shell init code, PATH entries, or aliases use `dis tools` subcommands. Each command upserts a named section delimited by `# BEGIN … import generated by dis config` / `# END … import generated by dis config` markers — so running the same command twice is safe (idempotent), and updating the content replaces the old section in place.

```bash
# Add a block to ~/rc/configs-generated/bash_init (sourced on interactive shell startup)
dis tools add-rc-init --name "Autojump" \
  --content '[[ -s ~/.autojump/etc/profile.d/autojump.sh ]] && source ~/.autojump/etc/profile.d/autojump.sh'

# Add a PATH export to ~/rc/configs-generated/bash_paths
dis tools add-rc-path --name "Mise path" \
  --content 'export PATH="$HOME/.local/share/mise/shims:$PATH"'

# Add an alias block to ~/rc/configs-generated/bash_aliases
dis tools add-rc-aliases --name "Notifier" \
  --content '[[ -s "${HOME}/.local/share/notifier/notifier_aliases" ]] && source "${HOME}/.local/share/notifier/notifier_aliases"'

# Wire ~/.bashrc to source a dotfiles .bashrc
dis tools add-home-rc --name "bashrc" \
  --content 'if [ -f /path/to/dotfiles/.bashrc ]; then . /path/to/dotfiles/.bashrc; fi'
```

`bash_paths` and `bash_aliases` are sourced by the wrapper before each installer, so PATH additions written by one installer are available to later ones in the same run. `bash_init` is sourced by `~/.bashrc` for interactive shells only.

---

## Commands

The `--distro` flag is optional on all commands if a [config file](#config-file) is present.

| Command | Description |
|---|---|
| `dis init` | Scaffold a workspace in the current directory |
| `dis install [--distro FILE]` | Install all packages in the distro |
| `dis install [--distro FILE] PKG` | Install a single package (skips dependency resolution) |
| `dis install [--distro FILE] --reinstall` | Re-run all installers, ignoring install state |
| `dis config [--distro FILE]` | Re-apply configs for all packages (skips install steps) |
| `dis config [--distro FILE] PKG` | Re-apply config for a single package |
| `dis plan [--distro FILE]` | Show the ordered install plan without executing |
| `dis search [--distro FILE] --regex PATTERN` | Search available packages by name |
| `dis list` | List all packages recorded as installed |
| `dis sync` | Update built-in packages from the latest release |
| `dis tools add-rc-init` | Upsert a section in `~/rc/configs-generated/bash_init` |
| `dis tools add-rc-path` | Upsert a section in `~/rc/configs-generated/bash_paths` |
| `dis tools add-rc-aliases` | Upsert a section in `~/rc/configs-generated/bash_aliases` |
| `dis tools add-home-rc` | Upsert a section in `~/.bashrc` |

---

## Config file

dis reads an optional config file from `~/.config/dis/config.yaml`. Values defined there are used as defaults for all commands — CLI flags always take precedence.

```yaml
# ~/.config/dis/config.yaml
distro: ~/dotfiles/dis/distros/home-server.yml
sources: ~/dotfiles/dis/packages
```

Supported keys:

| Key | Description |
|---|---|
| `distro` | Default path to the distro YAML file. Paths starting with `~/` are expanded to the home directory. |
| `sources` | Default value for `--sources` (overrides auto-detection of `${common_sources}`). |

With a config file in place you can omit `--distro` on every command:

```bash
dis install        # uses distro from config file
dis config         # same
dis plan           # same
```

---

## The `config` command

`dis config` re-runs every installer script with `DIS_INSTALL` **unset**, instructing scripts to perform only their configuration steps (copying config files, writing RC sections, applying `gsettings`, etc.) without re-running the full installation (downloading binaries, `apt install`, etc.).

This is useful when a config file has changed and you want to re-deploy it without reinstalling anything:

```bash
# Re-apply configs for all packages in the distro
dis config

# Re-apply config for a single package
dis config common/starship
```

`dis config` always runs regardless of install state and does **not** write install state afterwards.

### Writing installers that support `dis config`

Guard install-only steps behind `if [[ -n "${DIS_INSTALL:-}" ]]; then`. Configuration steps (file copies, `dis tools add-rc-*`, `gsettings`, etc.) are left outside the guard so they run under both commands.

```bash
### -- Manifest
### provides: common/starship
### distro: [all]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  # Install-only: download and install the binary
  curl -sS https://starship.rs/install.sh | sh

  # Wire into the shell (one-time setup)
  dis tools add-rc-init --name 'Starship' \
    --content $'eval "$(starship init bash)"'
fi

# Config: always re-deploy the config file
mkdir -p ~/.config/
cp "$DIS_CONFIG_FOLDER/starship/starship.toml" ~/.config/
```

| Mode | `DIS_INSTALL` | Install steps run? | Config steps run? |
|------|--------------|-------------------|-------------------|
| `dis install` | `"1"` | ✅ | ✅ |
| `dis config` | absent | ❌ | ✅ |

---

## Distro parameters and ${home}

The `parameters` block supports `${home}` which expands to the current user's home directory at load time:

```yaml
parameters:
  MY_PATH: ${home}/some/dir
```

---

## Scoped parameters

By default every entry in `parameters` is *global* — available to any installer that declares it in `requires_env`. If you want to restrict a parameter to a specific set of packages, add a `packages` list to the parameter definition:

```yaml
parameters:
  MY_PARAM:
    value: my-value
    packages: [mydis/package1, mydis/package2]
```

`MY_PARAM` is injected only when `mydis/package1` or `mydis/package2` runs.

### Mixing globals and scoped parameters

Both forms coexist freely in the same `parameters` block:

```yaml
parameters:
  GLOBAL_PARAM: global-value   # plain string — available to every package

  MY_PARAM:
    value: my-value
    packages: [mydis/package1, mydis/package2]   # only injected into package1 and package2

  ANOTHER:
    value: foo
    packages: [mydis/package2]                   # only injected into package2

packages:
  - hello/greet
  - mydis/package1
  - mydis/package2
```

### Resolution order

When dis builds the environment for a package it applies parameters in this order (later entries win on key conflicts):

1. Global `parameters` (plain string values).
2. Scoped `parameters` whose `packages` list includes this package.

So a scoped value always overrides a global with the same name.

---

## Built-in common packages

The `${common_sources}` token in `sources` expands to `~/.local/share/dis/packages`, which contains packages shipped with dis. Commonly used built-in packages include `common/bash-config`, `common/mise`, `common/go`, `common/python`, `common/node`, `common/docker`, and others.

Run `dis plan` to see the full list available for your distro.
