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

This installs the `dis` binary to `~/.local/bin` (or `INSTALL_DIR`). Installers, including the built-in common packages, live in git repos that `dis pull` fetches (see [Repos and dis pull](#repos-and-dis-pull)).

To set up a machine from a distro that lives in a git repo:

```bash
dis pull git@github.com:me/dotfiles.git --distro distros/laptop/laptop.yml
dis install
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

repos:
  dis:
    url: https://github.com/napicella/dis.git

sources:
  - .                      # this directory (dis reads dis.ws.yml here)
  - ${repos.dis}/packages  # built-in dis packages

packages:
  - hello/greet
```

- **`os`** — the target operating system (`ubuntu`, `amazon_linux`, or `all`).
- **`repos`** — git repos the sources live in, referenced as `${repos.<name>}`. See [Repos and dis pull](#repos-and-dis-pull).
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

### Bundles

A bundle is an installer with only a manifest: it `depends_on` the packages it groups and has no
steps of its own. Listing it in a distro file installs all of them.

```bash
### -- Manifest
### provides: bundle/containers
### depends_on: [common/docker, common/lazydocker]
### distro: [all]
### -- End
```

- Full runs (`dis install`, `dis config` without a package name) resolve dependencies, so they
  run the bundle's packages. With a package name, add `--with-deps`: `dis install bundle/containers`
  alone fails because the bundle's packages are not installed, and `dis config bundle/containers`
  alone runs only the empty bundle.
- Put a bundle where every distro that loads it can also see all its packages. dis rejects any
  loaded installer that depends on an unknown package, even if no distro lists it.
- A bundle can only include packages that have an installer for each OS it runs on.

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

The configs folder holds only files that installers deploy (copy, link or point
a tool at). `dis search configs .` lists everything an installer references
through `$DIS_CONFIG_FOLDER`, so helper scripts that installers `source` belong
elsewhere, e.g. in a `lib/` folder under the package root:

```bash
source "$DIS_PKG_ROOT/lib/toolbox-lib.sh"
```

`.sh` files without a manifest block are not treated as installers, so a `lib/`
folder under the root is safe.

---

## Environment variables available in installers

The following variables are set by dis in the installer's environment:

| Variable | Description |
|---|---|
| `DIS_PACKAGE` | Name of the package being installed, e.g. `common/mise`; `dis tools add-rc-*` records it as the section's owner |
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

**Example 2 — source with a dis.ws.yml** (e.g. the built-in packages, with the dis repo cloned at `~/dis`)

Given this `dis.ws.yml` in `~/dis/packages`:
```yaml
packages:
  - root: ./all
    configs: ./all/configs
```
and an installer at:
```
~/dis/packages/all/installers/00_bash_config.sh
```
dis sets:
```
DIS_PKG_ROOT      = /home/nicola/dis/packages/all
DIS_INSTALLER     = /home/nicola/dis/packages/all/installers/00_bash_config.sh
DIS_CONFIG_FOLDER = /home/nicola/dis/packages/all/configs
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

1. Creates `~/rc/configs-generated/` and ensures `bash_env`, `bash_paths`, `bash_aliases`, and `bash_init` exist.
2. Sources `bash_env`, `bash_paths` and `bash_aliases`, in that order, so exports and PATH additions from earlier installers propagate to the current one.
3. Prepends the `dis` binary directory to `PATH` so installers can call `dis tools ...` directly.
4. Runs the installer with `bash -e` (exit on error).

You do **not** need to add any `source` line to your installer scripts.

---

## RC helper tools

Installers that need to register exports, PATH entries, shell init code, or aliases use `dis tools` subcommands. Each command upserts a named section, so running the same command twice is safe (idempotent), and updating the content replaces the section in place.

```bash
# Add a block to ~/rc/configs-generated/bash_init (sourced on interactive shell startup)
dis tools add-rc-init --name "Autojump" \
  --content '[[ -s ~/.autojump/etc/profile.d/autojump.sh ]] && source ~/.autojump/etc/profile.d/autojump.sh'

# Add an export to ~/rc/configs-generated/bash_env
dis tools add-rc-env --name "Editor default" --content 'export EDITOR="${EDITOR:-vim}"'

# Prepend a dir to PATH in ~/rc/configs-generated/bash_paths (skipped if PATH already has it)
dis tools add-rc-path --name "Mise path" --path '$HOME/.local/share/mise/shims'

# Add an alias block to ~/rc/configs-generated/bash_aliases
dis tools add-rc-aliases --name "Notifier" \
  --content '[[ -s "${HOME}/.local/share/notifier/notifier_aliases" ]] && source "${HOME}/.local/share/notifier/notifier_aliases"'

# Remove a section a package no longer provides (no-op if absent)
dis tools rm-rc-section --file bash_aliases --name "Notifier"

# Wire ~/.bashrc to source a dotfiles .bashrc
dis tools add-home-rc --name "bashrc" \
  --content 'if [ -f /path/to/dotfiles/.bashrc ]; then . /path/to/dotfiles/.bashrc; fi'
```

`~/.bashrc` (through `~/rc/bash_config.sh`) sources `bash_env`, then `bash_paths`, then `bash_aliases`, and, in interactive shells only, `bash_init`. `bash_env` comes first so PATH entries can use the variables it exports. The wrapper sources the first three before each installer, so exports and PATH additions written by one installer are available to later ones in the same run. Debian and Ubuntu's bash also reads `~/.bashrc` for non-interactive commands run over SSH (`ssh host cmd`), which skip `bash_init`: whatever such commands need belongs in `bash_env` (exports, with `add-rc-env`) and `bash_paths` (PATH entries, with `add-rc-path --path`).

`add-rc-path` only takes `--path`; exports go to `bash_env` with `add-rc-env`.

Use `--path` rather than a hand-written `export PATH=...`: shells started from another shell (tmux, herdr) inherit PATH and source `bash_paths` again, and `--path` writes a guard so the dir is not added twice. `--path` can be repeated; the first one ends up first in PATH.

**Write rc sections outside the install guard.** They are config: `dis config` rewrites every section and records its owner, so a section written only under `DIS_INSTALL` is not refreshed by `dis config`.

### Generated files and the state file

The files in `~/rc/configs-generated/` are generated: dis renders them from a state file, `~/.local/share/dis/sections.yaml`, which holds each section's name, content and owner. Don't edit the generated files. Change the installer that writes the section and run `dis config` instead, or remove a section with `dis tools rm-rc-section`.

- **Order.** An updated section keeps its place in the file; a new one is appended.
- **Owner.** Run from an installer, a section is owned by the installer's package (`$DIS_PACKAGE`), shown in its marker as `[owned by common/mise]`. The last writer wins: writing a section replaces it and makes the writer its owner. `add-rc-aliases --owner PKG` locks the section (`[locked by PKG]`): only that package can overwrite or remove it.
- **Edits made outside dis.** dis records a hash of what it last wrote. If a generated file changed since, the next write backs it up as `FILE.bak-<timestamp>`, prints where, and regenerates it.
- **First run.** Without a state file, the first write imports the sections of the existing generated files (backing the files up first), and from then on renders them. A lost or corrupt state file is rebuilt the same way.

`~/.bashrc` is shared with other tools, so `add-home-rc` edits its marker block in place instead.

### Checking for drift

`dis doctor` reports drift dis doesn't catch on its own:

- lines in `~/.bashrc` outside dis sections (e.g. a tool's own PATH line), and a `~/.bashrc` that doesn't load `~/rc/bash_config.sh`
- generated files edited outside dis (with a diff), missing, or behind the state
- unguarded `export PATH=X:$PATH` lines in sections
- lines in the wrong generated file: exports and other non-PATH lines in `bash_paths` sections, PATH changes in `bash_env` sections
- orphan sections: owned by a package that is neither in the distro nor installed, e.g. left behind by a package removed from the distro
- PATH dirs that don't exist, each with the section or rc line that adds it and how to fix it
- source repos with uncommitted changes, unpushed commits, or fetched commits `dis pull` couldn't merge. Repos aren't fetched: run `dis pull` first to compare with the latest.

The report starts with the dis build running (also `dis --version`): commit, commit time, and `+modified` for a build with uncommitted changes. After pulling the dis repo, rebuild dis if that commit is older.

`dis doctor --fix` removes orphan sections (backing up their files) and regenerates generated files that are missing or behind the state. It never overwrites a generated file edited outside dis, and it only reports the lines in `~/.bashrc`: moving them would change when and in which shells they run.

---

## Config templates

Some config files are also written by other tools: a theming tool sets the theme name or the colors. Copying the file from the configs folder would lose those values. `dis tools render-config SRC DEST` renders SRC, a [Go template](https://pkg.go.dev/text/template), to DEST, and the template can keep values from DEST's current content:

- `{{ keep "PATH" DEFAULT }}`: the value at PATH in DEST (`theme-name`, `theme.name`, `palettes.custom.main_color`; see [Paths](#paths)), or DEFAULT when DEST does not exist or has no value there (null counts as no value). DEFAULT is a string, number or bool. The value is rendered as a literal in DEST's format: `"dark"` quoted and escaped, `3` and `true` as is, arrays and maps as JSON values, TOML inline tables (keys sorted) or YAML flow collections.
- `{{ keepTable "PATH" }}` (TOML only): a blank line, then the table at PATH as TOML text: the `[PATH]` header, its other keys, then its subtables with their own headers, each sorted by key. Empty when DEST has no such table. See [Optional tables](#optional-tables) for where to put it.

- **Naming.** Name a template after its destination plus `.tmpl` (`config.toml.tmpl`, `settings.json.tmpl`). The format comes from DEST, so the suffix changes nothing for `render-config`, but it keeps editors from validating the template as TOML or JSON, where every `{{ }}` is an error. The repo's `tombi.toml` skips `**/*.toml.tmpl`; to keep highlighting in VS Code, map `*.toml.tmpl` to TOML in `files.associations` (JSON templates are best mapped to plain text: VS Code's JSON checker can't be turned off per file).

```bash
# $DIS_CONFIG_FOLDER/ulauncher.json.tmpl has the line
#   "theme-name": {{ keep "theme-name" "dark" }}
dis tools render-config "$DIS_CONFIG_FOLDER/ulauncher.json.tmpl" ~/.config/ulauncher/settings.json

# $DIS_CONFIG_FOLDER/starship/starship.toml.tmpl has the lines
#   palette = {{ keep "palette" "custom" }}
#
#   [palettes.custom]
#   main_color = {{ keep "palettes.custom.main_color" "#A08AE2" }}
#   secondary_color = {{ keep "palettes.custom.secondary_color" "#A9E9B7" }}
dis tools render-config "$DIS_CONFIG_FOLDER/starship/starship.toml.tmpl" ~/.config/starship.toml
```

Keep each key of a table the template needs, as starship does, rather than the whole table with `keepTable`: on a new host `keepTable` renders nothing, and `palette = "custom"` would name a palette that doesn't exist.

- **Format.** It comes from DEST's extension: `.json`, `.toml`, `.yaml` or `.yml`. DEST is parsed only when the template calls `keep` or `keepTable`, so a template without them is a plain copy, for any file.
- **Errors.** A DEST that can't be parsed, or a template error (reported with its line and column in SRC), fails the command and leaves DEST untouched. The parse error comes first and says how to recover, naming the installer's package (`$DIS_PACKAGE`) when there is one: `cannot parse /home/me/.config/herdr/config.toml (line 3, column 8: toml: incomplete number): fix or delete it, then re-run dis config for common/herdr`.
- **Writing.** DEST is written atomically (a temp file in its directory, then a rename), its parent dirs are created, and it keeps its file mode (0644 for a new file). A DEST that is a symlink stays one: its target is written, and created when the link is dangling (a relative target is relative to the link's directory).
- **Re-rendering.** What `keep` and `keepTable` write is read back by the next run, so running it again (e.g. `dis config`) gives the same file.
- **Literal braces.** A `{{` that belongs to the config itself is written `{{ "{{" }}` in SRC.

### Paths

A path goes through two layers. The template's quotes, `"…"` or backticks, only make a Go string, like any other template argument. `keep` and `keepTable` then parse the characters they receive: `.` separates keys, and double quotes group a key, dots included, with `\"` and `\\` as escapes inside, as in TOML dotted keys.

| Template | `keep` receives | Keys looked up |
|---|---|---|
| `"palettes.custom.main_color"` | `palettes.custom.main_color` | `palettes` → `custom` → `main_color` |
| `"workbench.colorTheme"` | `workbench.colorTheme` | `workbench` → `colorTheme` (not the flat key) |
| `` `"workbench.colorTheme"` `` | `"workbench.colorTheme"` | `workbench.colorTheme` |
| `"\"workbench.colorTheme\""` | `"workbench.colorTheme"` | same, escaped instead of backticks |
| `` `"[python]"."editor.tabSize"` `` | `"[python]"."editor.tabSize"` | `[python]` → `editor.tabSize` |

With VS Code's `settings.json` holding `{ "workbench.colorTheme": "Nord" }`:

```
{{ keep "workbench.colorTheme" "Default Dark+" }}
{{ keep `"workbench.colorTheme"` "Default Dark+" }}
```

the first line renders `"Default Dark+"`, as the file has no `workbench` object, and the second `"Nord"`. Write a path that needs quotes in backticks: the quotes reach `keep` as typed, with no Go escaping to get right. A path without quotes parses as plain dotted keys. An unquoted key can't hold whitespace, so a stray space (`x. y`) fails instead of looking up the key ` y`; quote a key whose name has spaces (`` `"weird key"` ``). An unquoted key with whitespace, an unterminated quote, an empty key (`a..b`, or a leading or trailing dot) or anything but `.` after a closing quote fails the render with the path in the error.

### Optional tables

A table only the other tool writes, like herdr's `[theme.custom]` colors, goes in the template as `{{- keepTable "PATH" }}` on the line after the last line before it, followed by a blank line and the next section:

```toml
# text = "#cdd6f4"
{{- keepTable "theme.custom" }}

[terminal]
```

`{{-` and `-}}` are standard Go template [whitespace trimming](https://pkg.go.dev/text/template#hdr-Text_and_spaces): they drop the spaces and newlines before or after the action (the space after `{{-` is required). Here `{{-` drops the newline before it, and since `keepTable` starts with a blank line there is exactly one blank line on each side of the table, whether DEST has it or not. Without the table in DEST:

```toml
# text = "#cdd6f4"

[terminal]
```

With it:

```toml
# text = "#cdd6f4"

[theme.custom]
accent = "#ff79c6"

[theme.custom.dark]
panel_bg = "#1e1e2e"

[terminal]
```

The bare form, `{{ keepTable "theme.custom" }}` on its own line without `{{-`, keeps the newline before the action, so the table gets two blank lines before it, and a DEST without it leaves two blank lines in a row: use the `{{-` form.

The table is written from DEST's parsed values, so comments and blank lines inside it (between its keys or subtables) are not kept: only its header, keys and subtables are. TOML tables are unordered, so keys and subtables are written sorted, and how a table was written in DEST is not kept either: an inline table (`dark = { panel_bg = "#1e1e2e" }`) or dotted keys (`dark.panel_bg = ...`) come out as a `[theme.custom.dark]` subtable, an array of tables (`[[...]]`) as an inline array of inline tables. The data is the same. `keepTable` on an array of tables itself fails (`an array of tables, not a table; use keep`): `keep` renders it as an inline array. Comments elsewhere in the template are text and are written as they are.

For a default table when DEST has none, use `with` and `else`:

```toml
# text = "#cdd6f4"
{{- with keepTable "theme.custom" }}{{ . }}{{ else }}

[theme.custom]
accent = "#f5c2e7"
{{- end }}

[terminal]
```

---

## Commands

The `--distro` flag is optional on all commands if a [config file](#config-file) is present.

| Command | Description |
|---|---|
| `dis init` | Scaffold a workspace in the current directory |
| `dis install [--distro FILE]` | Install all packages in the distro |
| `dis install [--distro FILE] PKG` | Install a single package. Fails before running it if a package it depends on is not recorded as installed, suggesting `--with-deps`. The dependencies of an installed package are not checked, nor is a package that is already installed (it is skipped) |
| `dis install [--distro FILE] --no-deps-check PKG` | Install a single package without checking its dependencies (e.g. installed outside dis) |
| `dis install [--distro FILE] --with-deps PKG` | Install a package and everything it depends on, dependencies first (e.g. a whole bundle) |
| `dis install [--distro FILE] --reinstall` | Re-run all installers, ignoring install state |
| `dis config [--distro FILE]` | Re-apply configs for all packages (skips install steps) |
| `dis config [--distro FILE] PKG` | Re-apply config for a single package |
| `dis config [--distro FILE] --with-deps PKG` | Re-apply config for a package and everything it depends on |
| `dis plan [--distro FILE]` | Show the ordered install plan without executing |
| `dis search packages REGEX [--distro FILE]` | Search the names of the available packages, installed or not |
| `dis search installers REGEX [--package REGEX] [--distro FILE]` | Search installer lines, e.g. to find which package defines an alias |
| `dis search configs REGEX [--package REGEX] [--distro FILE]` | Search the paths of the config files installers reference |
| `dis search ... --json` | Print any search as a JSON array of `{package, path, line, text}` |
| `dis edit [--distro FILE] PKG` | Open the package's installer and config files in `$DIS_EDITOR`, `$VISUAL`, `$EDITOR` or `vi` (first set) |
| `dis edit [--distro FILE] PKG --installer` / `--configs` | Open only the installer, or only the config files |
| `dis edit [--distro FILE] PKG --apply` | Open, then re-apply the package's config once the editor exits successfully |
| `dis list` | List all packages recorded as installed |
| `dis --version` | Show the commit dis was built from |
| `dis doctor [--distro FILE] [--fix]` | Check the rc files, install state and source repos for drift; `--fix` removes orphan sections and regenerates missing or outdated generated files |
| `dis sources [--distro FILE] [--json]` | List the distro's resolved source directories, with their repo and package count |
| `dis pull GIT-URL [--distro FILE] [--path DIR]` | Clone a distro repo and every repo it declares, and set it as the default distro |
| `dis pull` | Clone or fast-forward the repos of the configured distro |
| `dis tools add-rc-env` | Upsert a section in `~/rc/configs-generated/bash_env` (exports) |
| `dis tools add-rc-path` | Upsert a section in `~/rc/configs-generated/bash_paths` (`--path DIR` for PATH entries) |
| `dis tools add-rc-init` | Upsert a section in `~/rc/configs-generated/bash_init` |
| `dis tools add-rc-aliases` | Upsert a section in `~/rc/configs-generated/bash_aliases` |
| `dis tools rm-rc-section --file FILE --name NAME` | Remove a section from a generated file (`bash_env`, `bash_paths`, `bash_aliases` or `bash_init`) |
| `dis tools add-home-rc` | Upsert a section in `~/.bashrc` |
| `dis tools render-config SRC DEST` | Render the template SRC to DEST, keeping values other tools wrote in DEST (see [Config templates](#config-templates)) |

---

## Searching packages

`dis search` looks through every package the distro's sources provide, installed
or not. A subcommand says what to search; each takes exactly one pattern.
Patterns are [Go regular expressions](https://pkg.go.dev/regexp) and are not
anchored: `git` matches anything containing `git`.

```bash
# Packages whose name contains "git"
dis search packages git

# Which package defines the `status` alias?
dis search installers status
dis search installers status --package git   # only in packages named *git*
dis search installers 'status.*git'          # both terms on the same line

# Config files of a tool (the source files in the repo, not the deployed copies)
dis search configs . --package starship
dis search configs tmux                      # config paths containing "tmux"
```

Every subcommand prints the same shape, one aligned row per result: the package,
then a path.

```
$ dis search packages git
common/git       …/installers/03_git.sh
$ dis search installers status
common/git       …/installers/03_git.sh:29  alias status='git status'
$ dis search configs . --package starship
common/starship  …/configs/starship/starship.toml.tmpl
```

- `packages` and `installers` print the installer path. `installers` appends
  `:line` to it and adds the matching text as a third column; the manifest header
  is skipped.
- `configs` prints a file or folder the installer references as
  `$DIS_CONFIG_FOLDER/<path>`, and matches the pattern against that path. The path
  stops at the first variable (`$DIS_CONFIG_FOLDER/backgrounds/$THEME` gives the
  `backgrounds` folder), comment lines are ignored and missing paths are skipped.
- `--package REGEX` (`installers` and `configs`) only looks at the packages whose
  name matches.
- The command exits non-zero when nothing matches, and when the subcommand or
  the pattern is missing.

To see what is installed on this machine use `dis list`; to see where packages
come from use `dis sources`.

### JSON output

`--json` prints the same results as a JSON array, for scripts. Every object has
`package` and `path`; `line` and `text` are set only for `installers`. An empty
search prints `[]` (and still exits non-zero).

```bash
dis search installers status --json
# [{"package": "common/git", "path": "…/03_git.sh", "line": 29, "text": "alias status='git status'"}]

# Edit a tool's config, then re-deploy it
vim $(dis search configs . --package starship --json | jq -r '.[].path')
dis config common/starship

# Re-apply the package that defines an alias
dis config "$(dis search installers 'alias status=' --json | jq -r '.[0].package')"
```

---

## Config file

dis reads an optional config file from `~/.config/dis/config.yaml`. Values defined there are used as defaults for all commands — CLI flags always take precedence.

```yaml
# ~/.config/dis/config.yaml
distro: ~/dotfiles/distros/home-server/home-server.yml
```

Supported keys:

| Key | Description |
|---|---|
| `distro` | Default path to the distro YAML file. Paths starting with `~/` are expanded to the home directory. `dis pull` sets it. |

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

# Re-apply config for a package and everything it depends on, e.g. a bundle
dis config --with-deps bundle/shell
```

`dis config` always runs regardless of install state and does **not** write install state afterwards.

### Writing installers that support `dis config`

Guard install-only steps behind `if [[ -n "${DIS_INSTALL:-}" ]]; then`. Configuration steps (file copies, `dis tools add-rc-*`, `gsettings`, etc.) are left outside the guard so they run under both commands. Put rc sections before the guard when it can `exit` early (e.g. "already installed").

```bash
### -- Manifest
### provides: common/starship
### distro: [all]
### -- End

# Config: wire into the shell (rc sections are config)
dis tools add-rc-init --name 'Starship' \
  --content $'command -v starship &> /dev/null && eval "$(starship init bash)"'

if [[ -n "${DIS_INSTALL:-}" ]]; then
  # Install-only: download and install the binary
  curl -sS https://starship.rs/install.sh | sh
fi

# Config: always re-deploy the config file, keeping the colors a theming tool
# set (starship.toml is a template, see Config templates)
dis tools render-config "$DIS_CONFIG_FOLDER/starship/starship.toml.tmpl" ~/.config/starship.toml
```

| Mode | `DIS_INSTALL` | Install steps run? | Config steps run? |
|------|--------------|-------------------|-------------------|
| `dis install` | `"1"` | ✅ | ✅ |
| `dis config` | absent | ❌ | ✅ |

### One-time migrations

Sometimes a config step only exists to clean up after an older version of a package: a renamed
package releasing its old rc lock, a stale binary or unit file, a removed rc section. Mark such a
block with the date it was added, so it can be deleted once every host has applied it:

```bash
# MIGRATION(2026-10-01): one-time cleanup; drop once every host has run dis config since then.
# <why the cleanup is needed>
if <old state is present>; then
  <cleanup>
fi
```

Keep the block safe to run again, like any config step: check for the old state first, so it is a
no-op once the cleanup is done. List the blocks still waiting to be removed with:

```bash
dis search installers 'MIGRATION\('
```

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

## Repos and dis pull

A distro declares the git repos its sources live in:

```yaml
repos:
  dotfiles: { url: git@github.com:me/dotfiles.git }                  # cloned to ~/dotfiles
  dis:      { url: https://github.com/napicella/dis.git, path: ~/github/dis, ref: main }
sources:
  - ${repos.self}/tools        # the repo that contains this distro file
  - ${repos.dotfiles}/bashrc
  - ${repos.dis}/packages
```

- **`url`** (required) is passed to `git clone` as is.
- **`path`** defaults to `~/<name>`. `~` and `${home}` are expanded, and relative paths are relative to the home directory.
- **`ref`** is an optional branch or tag to clone; the default is the remote's default branch.
- **`${repos.self}`** is implicit: the root of the git repo that contains the distro file, or the distro file's folder if it isn't in a git repo. `self` can't be declared.
- `${repos.<name>}` works in `sources`, precondition scripts and `parameters`. Referencing an undeclared repo, or one that isn't cloned yet, is an error.

`dis pull` fetches them:

```bash
dis pull GIT-URL [--distro FILE] [--path DIR]   # first time on a machine
dis pull                                        # later: update the configured distro's repos
```

With a URL, dis clones that repo (default `~/<repo-name>`) and finds the distro file in it: `--distro` is relative to the repo, and can be omitted when the repo contains exactly one distro. It then clones or updates every declared repo and writes `distro:` to `~/.config/dis/config.yaml`.

Repos that already exist are fetched and fast-forwarded only when the work tree is clean, on a branch that tracks an upstream, and not diverged. Otherwise they are skipped with the reason, so local work is never touched. `install`, `config` and `plan` never touch the network and read the local clones, so local edits take effect right away.

dis runs the `git` CLI, so your SSH config, keys and credential helpers apply.

## Built-in common packages

The dis repo's `packages/` folder contains packages shipped with dis. Declare the dis repo and add `${repos.dis}/packages` to `sources` to use them. Commonly used built-in packages include `common/bash-config`, `common/mise`, `common/go`, `common/python`, `common/node`, `common/docker`, and others.

Run `dis plan` to see the full list available for your distro.
