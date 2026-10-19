# dis

`dis` is a distro installer CLI — it reads a distro YAML file, resolves installer manifests and their dependencies, and runs each installer in topological order.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/install.sh | bash
```

To install to a custom location:

```bash
INSTALL_DIR=/usr/local/bin bash <(curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/install.sh)
```

### GitHub SSH key for a new host

`dis pull` runs `git`, so private GitHub repos cloned over SSH need a key on the host first:

```bash
curl -fsSL https://raw.githubusercontent.com/napicella/dis/main/github-ssh.sh | bash
# or pick the key name (defaults to github_ed25519): ... | bash -s -- --key ~/.ssh/my_github
```

It pins GitHub's host keys in `~/.ssh/known_hosts`, adds a `Host github.com` block to `~/.ssh/config`,
creates `~/.ssh/github_ed25519` unless it exists, prints the public key to add at
https://github.com/settings/ssh/new, and checks that GitHub accepts it. Safe to re-run: it stops
right away on a host that's already set up.

## Usage

```bash
# Show the ordered install plan without executing anything
dis plan --distro ~/dotfiles/distros/laptop/laptop.yml

# Run the installation
dis install --distro ~/dotfiles/distros/laptop/laptop.yml
```

## Release

```bash
make release VERSION=v0.1.0
```

This tags the commit and pushes the tag, which triggers the GitHub Actions release workflow to build and publish binaries.
