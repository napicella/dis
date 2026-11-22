# dis

`dis` is a distro installer CLI — it reads a distro YAML file, resolves installer manifests and their dependencies, and runs each installer in topological order.

Full documentation (distro files, installers, commands): [docs/user-guide.md](docs/user-guide.md).

## Set up a host

Given a distro repo and the distro file in it, one command installs dis, gives the host a GitHub SSH
key, then runs `dis pull` and `dis install`:

```bash
curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/install.sh | bash -s -- \
  git@github.com:me/dotfiles.git distros/home-server/home-server.yml
```

You add the host's key on GitHub, and, if an installer needs a secret, paste the Bitwarden
Secrets Manager token once (see [Secrets](docs/user-guide.md#secrets)). Installers that still
prompt on their own (e.g. for a token they fetch secrets with) ask on the terminal, as with a plain
`dis install`.

## Step by step

The same, one step at a time.

1. Install dis to `~/.local/bin`:

   ```bash
   curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/install.sh | bash
   # or to another dir:
   INSTALL_DIR=/usr/local/bin bash <(curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/install.sh)
   ```

2. Give the host its own GitHub SSH key, needed to clone private repos over SSH. Safe to re-run: it
   stops at once on a host GitHub already accepts.

   ```bash
   curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/github-ssh.sh | bash
   # or pick the key (defaults to ~/.ssh/github_ed25519): ... | bash -s -- --key ~/.ssh/my_github
   ```

   It adds GitHub's host keys to `~/.ssh/known_hosts` and a `Host github.com` block to
   `~/.ssh/config`, creates the key unless it exists, then prints the public key to add at
   https://github.com/settings/ssh/new and waits until GitHub accepts it.

3. Clone the distro repo and the repos it declares, and make it the default distro:

   ```bash
   dis pull git@github.com:me/dotfiles.git --distro distros/home-server/home-server.yml
   ```

4. Install it:

   ```bash
   dis install
   ```

## Everyday use

After `dis pull`, commands use the default distro (`~/.config/dis/config.yaml`); `--distro FILE`
picks another.

```bash
dis pull                 # update the distro's repos
dis plan                 # show the ordered install plan, without running anything
dis install              # install what isn't installed yet
dis config               # re-apply every package's config (skips install steps)
dis config common/git    # re-apply one package's config
```

See [Commands](docs/user-guide.md#commands) for the rest.

## Releasing dis

```bash
make release VERSION=v0.1.0
```

This tags the commit and pushes the tag, which triggers the GitHub Actions release workflow to build and publish binaries.
