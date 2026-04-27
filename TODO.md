# TODO

## common/bash-config installer

Add a `common/bash-config` installer in `packages/all/installers/` that sets up
the standard `~/rc/configs-generated` sourcing block in `~/.bashrc`.

The installer should append the following to `~/.bashrc` (idempotently):

```bash
# source generated configs if present
if [ -d ~/rc/configs-generated ]; then
    source ~/rc/configs-generated/bash_paths
    source ~/rc/configs-generated/bash_aliases
    source ~/rc/configs-generated/bash_init
fi
```

This makes `common/bash-config` the default bash setup that any distro can
include by adding it to `packages:` — no need to copy the block into every
distro-specific bashrc installer.
