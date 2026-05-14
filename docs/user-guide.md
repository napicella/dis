# dis — User Guide

## What is dis?

`dis` is a CLI for automating machine setup. You describe *what* to install in a distro YAML file and *how* to install it in shell scripts called installers. `dis` resolves dependencies between installers and runs them in the correct order, skipping anything already done.

---

## Install

```bash
export GITHUB_TOKEN=ghp_...
curl -fsSL \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  https://raw.githubusercontent.com/napicella/dis/main/install.sh \
  | GITHUB_TOKEN="${GITHUB_TOKEN}" bash
```

This installs the `dis` binary to `~/.local/bin` and downloads the built-in common packages to `~/.local/share/dis/packages`.

To keep the built-in packages up to date:

```bash
dis sync --token "$GITHUB_TOKEN"
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
  - .               # this directory (dis reads dis.ws.yml here)
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
source "$DIS_BINDING"

echo "Installing with MY_PARAM=${MY_PARAM}"
```

The manifest block tells dis everything it needs to know. The `source "$DIS_BINDING"` line loads the dis helper functions.

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

Write `KEY=value` lines to `$DIS_EXPORTS_FILE` (or use the `dis_export` helper):

```bash
GID_DOCKER=$(getent group docker | cut -d: -f3)
dis_export GID_DOCKER "$GID_DOCKER"
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

After `source "$DIS_BINDING"`, the following variables are set:

| Variable | Description |
|---|---|
| `DIS_PKG_ROOT` | The `root` declared in `dis.ws.yml` for this package, or the source directory itself when no workspace file is present |
| `DIS_CONFIG_FOLDER` | Configs directory (set from `dis.ws.yml`; empty if not declared) |
| `DIS_INSTALLER` | Absolute path to the installer script |
| `DIS_DISTRO` | OS name from the distro YAML |
| `DIS_EXPORTS_FILE` | Temp file to write `KEY=value` exports for downstream installers |

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
source "$DIS_BINDING"

GID_DOCKER=$(getent group docker | cut -d: -f3)
dis_export GID_DOCKER "$GID_DOCKER"
```

`consumer.sh` imports it using the qualified `pkg:VAR` syntax:
```bash
### -- Manifest
### provides: myapp/containers
### depends_on: [myapp/docker]
### requires_env: [myapp/docker:GID_DOCKER]
### -- End
source "$DIS_BINDING"

echo "Docker GID is: ${GID_DOCKER}"
```

dis injects `GID_DOCKER` (the bare name, without the `myapp/docker:` prefix) into the consumer's environment.

---

## Helper functions (from binding.sh)

```bash
# Add a line to ~/.local/share/bash_init (sourced on shell startup)
bashrc_init_add "label" 'your shell code here'

# Add a PATH export
bashrc_path_add "label" 'export PATH="$HOME/.local/bin:$PATH"'

# Add an alias
bashrc_aliases_add "label" 'alias ll="ls -la"'

# Export a value for downstream installers
dis_export KEY value
```

---

## Commands

| Command | Description |
|---|---|
| `dis init` | Scaffold a workspace in the current directory |
| `dis install --distro FILE` | Install all packages in the distro |
| `dis install --distro FILE --reinstall` | Re-run all installers, ignoring install state |
| `dis run --distro FILE PKG` | Run a single installer (skips dependency resolution) |
| `dis run --distro FILE PKG --reinstall` | Re-run even if already installed |
| `dis plan --distro FILE` | Show the ordered install plan without executing |
| `dis list` | List all packages recorded as installed |
| `dis sync --token TOKEN` | Update built-in packages from the latest release |

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
