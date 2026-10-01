---
name: dis
description: Work with dis, the distro installer CLI that installs and configures a machine from installer scripts listed in a distro YAML file. Use when the user asks to find, read, change or add a dis package or installer ("update mise in dis", "create an installer for dis that does X", "which dis package sets up Y", "add X to my distro"), or asks about dis commands, distro files, sources or rc sections.
---

# dis

dis runs installer scripts (bash, with a manifest header) that live in the
**sources** a **distro file** declares. The distro file is the single source of
truth for where packages are: never assume a repo or a directory, ask dis.

For anything this skill does not cover, `dis <command> --help` is
authoritative, then `docs/user-guide.md` in the dis repo
(`dis sources` shows where the dis repo is: the source of repo `dis`).

## Orient first

```bash
dis sources                   # sources of the current distro: declared, path, repo, package count
dis search packages .         # every package available (installed or not), with its installer path
dis list                      # packages recorded as installed on this machine
dis plan                      # what 'dis install' would run, in order
```

- The distro file comes from the `distro:` key of `~/.config/dis/config.yaml`,
  set by `dis pull`. Every command takes `--distro FILE` to target another host's distro.
- Each host usually has its own distro file. When a change should reach other
  hosts, their distro files matter too (see "Add it to distro files").

## Find a package

```bash
dis search packages mise --json                     # by name (Go regexp, unanchored)
dis search installers 'alias pull' --json           # by installer content -> package, file, line
dis search configs . --package starship --json      # config files a package deploys
```

Use `--json` and read `path` from it: the paths are resolved, so there is no
need to know which repo a package is in.

**One name, several OS variants.** dis only loads packages whose `distro:`
matches the distro file's `os`, so a search shows just this host's variant.
Before changing a package, find every variant across all sources:

```bash
grep -rl --include='*.sh' '^### provides: common/mise$' $(dis sources --json | jq -r '.[].path')
```

A fix usually belongs in all variants (e.g. both the `ubuntu` and
`amazon_linux` installer of `common/mise`); say so if you only change one.

## Read and change a package

Open the files from the search output and edit them directly with your file
tools. **Never run `dis edit`**: it opens an interactive editor and blocks.

Before changing an installer, read it whole and match its style: how it guards
install-only steps, which rc helpers it uses, how it copies configs.

## Create a package

1. **Pick the source.** Run `dis sources` and choose with the user,
   unless the request makes it obvious (a variant of an existing package goes
   next to it; "like the X package" goes where X is). Generic, reusable
   packages and personal ones usually live in different sources; ask rather
   than guess.
2. **Mirror a neighbour.** Look at two packages already in that source and copy
   their layout and naming. Sources differ: some have a `dis.ws.yml` that maps
   installer roots to a configs folder (`DIS_CONFIG_FOLDER` is set only then),
   some are walked whole. Installer file names follow the neighbours (e.g.
   `NN_name.sh` or `name-install.sh`); the `NN_` prefix is only for sorting,
   run order comes from `depends_on` and the distro's `packages:` list.
3. **Write the installer** from [references/installer.md](references/installer.md):
   manifest, install-only vs config steps, rc helpers.
4. **Add it to distro files** (below), then verify (below).

## Add it to distro files

A package runs only when a distro file lists it under `packages:` (or another
listed package depends on it). Ask the user which hosts should get it. The
current distro file is in `~/.config/dis/config.yaml`; other distro files are
YAML files with a `sources:` key, usually in the same repos:

```bash
grep -rl --include='*.yml' '^sources:' $(dis sources --json | jq -r '.[].path' | xargs -n1 dirname | sort -u)
```

Ignore test fixtures and sandboxes in the results (`tests/`, `scratch/`).

## Verify

In order, stopping at the first failure:

1. `bash -n <installer>`
2. `dis plan` lists the package where expected, and `dis search packages <name>`
   finds it (a manifest error hides it).
3. Dry run in a throwaway `$HOME`, so the real rc files are not touched. Put
   the dis binary on `PATH` and set what the installer reads:
   ```bash
   F=$(mktemp -d); mkdir -p "$F/bin"; cp "$(command -v dis)" "$F/bin/"
   HOME=$F PATH=$F/bin:$PATH DIS_INSTALLER=<installer> DIS_CONFIG_FOLDER=<configs dir> \
     bash -e <installer>
   # inspect $F/rc/configs-generated/*, run it again to check it is idempotent
   ```
   Skip steps that need root or the network, or run them in the Docker
   sandbox (`scratch/run-sandbox.sh` in the dis repo) instead.
4. `dis config <name>` or `dis install <name>` on the real machine changes the
   user's environment: only with their OK.

## Commit

A change often spans several repos (the dis repo, a dotfiles repo, a distro
repo): one commit per repo. Commit only when asked.
