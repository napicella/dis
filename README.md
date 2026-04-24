# dis

`dis` is a distro installer CLI — it reads a distro YAML file, resolves installer manifests and their dependencies, and runs each installer in topological order.

## Install

Set your GitHub personal access token first:

```bash
export GITHUB_TOKEN=ghp_...
```

Then install:

```bash
curl -fsSL \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  https://raw.githubusercontent.com/napicella/dis/main/install.sh \
  | GITHUB_TOKEN="${GITHUB_TOKEN}" bash
```

To install to a custom location:

```bash
INSTALL_DIR=/usr/local/bin bash <(
  curl -fsSL -H "Authorization: Bearer ${GITHUB_TOKEN}" \
    https://raw.githubusercontent.com/napicella/dis/main/install.sh
)
```

## Usage

```bash
# Show the ordered install plan without executing anything
dis plan --distro ~/dotfiles/distros/home-server/home-server.yml

# Run the installation
dis install --distro ~/dotfiles/distros/home-server/home-server.yml
```

## Release

```bash
make release VERSION=v0.1.0
```

This tags the commit and pushes the tag, which triggers the GitHub Actions release workflow to build and publish binaries.
