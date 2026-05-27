# dis — Architecture Overview

This document describes the internal structure of `dis` for contributors. It focuses on concepts, responsibilities, and data flow rather than implementation details, which are best read directly in the source.

---

## Core concepts

### Distro

A distro is a YAML file that describes a machine configuration: which OS it targets, where to find installer scripts (sources), which packages to install, and static configuration parameters. A distro file is the entry point for all `dis` commands.

### Source

A source is a directory that `dis` walks to discover installer scripts. Sources are listed in the distro YAML. Each source may optionally contain a `dis.ws.yml` workspace file.

### Workspace (dis.ws.yml)

A workspace file lives in a source directory and tells `dis` how to scope the walk. It declares a list of package roots (subdirectories) and, optionally, a configs directory for each. When a workspace file is present, `dis` only walks the declared roots rather than the whole source tree. This allows a single source directory to contain multiple independent packages, each with its own configs, without them interfering with each other.

When no workspace file is present, `dis` walks the entire source directory — this is the simple case for single-tool repositories.

### Manifest

A manifest is a structured comment block at the top of an installer `.sh` file. It declares the package's name (`provides`), the OS it applies to (`distro`), its dependencies (`depends_on`), the env vars it needs (`requires_env`), and the values it exports (`exports_env`). The manifest is parsed at load time; it is never executed.

### Package

A package is the combination of a manifest and the installer script it came from. Packages are identified by their `provides` name (e.g. `common/go`). A package may depend on other packages; `dis` resolves the full dependency graph and runs installers in topological order.

### Install context

The install context is the runtime state for a single `dis install` or `dis run` invocation. It holds the resolved package graph, the parameters map (static params + generator output + exported values from prior installers), and the distro configuration. It is the central object passed between the loader, the dependency resolver, and the installer runner.

### Exports cache

When an installer exports values, those values are written both into the current install context's parameters map and into a persistent cache on disk (`~/.local/share/dis/exports-cache.txt`). On future runs, when `dis install` or `dis run` is started with `NewInstallContextWithCache`, the cache is loaded upfront so that packages depending on exported values can succeed even when the exporting package is skipped as already-installed.

---

## Data flow

### `dis install`

1. **Load** — the distro YAML is parsed, sources are walked, manifests are collected, and the package dependency graph is built. Parameters from the distro YAML are loaded into the context (including `${home}` expansion). The exports cache is loaded into the parameters map.
2. **Generators** — config generator scripts are run in order; their `KEY=VALUE` stdout output is merged into the parameters map.
3. **Preconditions** — precondition scripts are run with their declared parameters; any failure aborts the installation.
4. **Resolve** — the full ordered list of manifests to install is computed by topological sort of the dependency graph for the declared packages.
5. **Run** — each installer is executed in order via the wrapper script. If a package is already recorded as installed, it is skipped (but the cache may have already populated its exports). After each successful install, any exported values are merged into the parameters map and the cache, and the package is recorded as installed.

### `dis run`

Same as `dis install` but targets a single named package, skipping dependency resolution. Generators and preconditions still run. Uses the same exports cache so exported values from previously-installed packages are available.

### `dis plan`

Same loading and resolution as `dis install` but nothing is executed. Prints the ordered list of installers with their paths and any `requires_env` entries.

---

## File layout

```
cmd/                  CLI commands (cobra); thin layer over internal/dis
internal/dis/
  types.go            Data types: DistroConfig, Manifest, WorkspaceConfig, etc.
  loader.go           Walk sources, parse dis.ws.yml, collect manifests
  parser.go           Parse manifest header blocks from .sh files
  pkgmng.go           Package dependency graph and topological sort
  context.go          InstallContext construction and parameter resolution
  installer.go        Run generators, preconditions, and installer scripts
  state.go            Installed-packages state file and exports cache (XDG)
  wrapper.sh          Shell wrapper sourced before every installer run
  binding.go          Embeds wrapper.sh and writes it to a temp file
  binding.sh          Tombstone — no longer used (see wrapper.sh)
  template.go         Installer script template (used by dis init)
internal/tools/
  rc.go               AddRCSection — upsert logic for generated RC files
  gnome_shortcut.go   GNOME keyboard shortcut helper
packages/             Built-in installer packages shipped with dis
  dis.ws.yml          Workspace file scoping the packages directory
  all/                Installers for all distros
  ubuntu/             Ubuntu-specific installers
  amazon_linux/       Amazon Linux-specific installers
tests/                Integration tests (Docker-based, build tag: integration)
docs/                 Documentation
```

---

## The wrapper contract

Every installer script is run inside a wrapper (`wrapper.sh`) written to a temp file by dis before each installer run. The wrapper:

1. Creates `~/rc/configs-generated/` and ensures `bash_paths`, `bash_init`, and `bash_aliases` exist.
2. Sources `bash_paths` and `bash_aliases` so that PATH additions written by earlier installers in the same run are available.
3. Prepends the dis binary directory to `PATH` so installers can call `dis tools ...` directly.
4. `exec`s the installer with `bash -e` (exit on error).

`bash_init` is intentionally **not** sourced by the wrapper — it contains interactive-session code that must only run in shells, not in non-interactive installer processes.

### RC helper commands

Installers that need to register shell initialization, PATH entries, or aliases use `dis tools` subcommands:

| bash function (old) | dis command (new) |
|---|---|
| `bashrc_init_add "NAME" "CONTENT"` | `dis tools add-rc-init --name NAME --content CONTENT` |
| `bashrc_path_add "NAME" "CONTENT"` | `dis tools add-rc-path --name NAME --content CONTENT` |
| `bashrc_aliases_add "NAME" "CONTENT"` | `dis tools add-rc-aliases --name NAME --content CONTENT` |
| `__bashrc_home_add "NAME" "CONTENT"` | `dis tools add-home-rc --name NAME --content CONTENT` |

All four commands use the same upsert logic (`internal/tools/rc.go`): sections are delimited by `# BEGIN <name> import generated by dis config` / `# END <name> import generated by dis config`. If a section with the same name already exists and the content is identical, it is left unchanged. If the content differs, the old section is replaced.

The `DIS_CONFIG_FOLDER`, `DIS_PKG_ROOT`, `DIS_INSTALLER`, `DIS_DISTRO`, and `DIS_EXPORTS_FILE` variables are set by `dis` in the installer's environment before the wrapper runs.

---

## Parameter resolution

When `envForInstaller` resolves a package's `requires_env` list, it consults the parameters map in this way:

- Bare names are looked up directly (from distro parameters or generator output).
- Qualified names (`pkg:VAR`) are looked up under that exact key, which was populated either when the exporting package ran or from the exports cache.
- Glob patterns (`FOO_*`, `pkg:PREFIX*`) inject all matching keys.

If a required value is not found, the error message names the missing key and the package that should have exported it, so the cause is immediately actionable.

---

## State and idempotency

`dis` records installed packages in `~/.local/share/dis/installed.txt`. Before running an installer, it checks this file and skips the package if already present. The `--reinstall` flag bypasses this check and resets the recorded state after a successful run.

The exports cache (`~/.local/share/dis/exports-cache.txt`) is a companion file that stores all `pkg:KEY=value` pairs ever exported. It is read on startup (when using `NewInstallContextWithCache`) and written on each successful export, using a read→merge→rewrite strategy to avoid stale entries.

---

## Testing

Unit tests live in `internal/dis/` alongside the code they test (plain `go test`). Integration tests live in `tests/` under the `integration` build tag and require a pre-built binary (`DISGO_BIN`) and Docker. Each integration test spins up a fresh container, copies the binary (as `dis`) and testdata in, runs dis commands, and verifies observable side effects (sentinel files, command output, RC file contents). The test image is built from `tests/testdata/Dockerfile`.
