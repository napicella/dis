#!/usr/bin/env bash
### -- Manifest
### provides: sample/hello
### depends_on: [common/os-libs]
### distro: [ubuntu]
### -- End

echo "==> sample/hello installer running"

# Register a PATH entry — appears in ~/rc/configs-generated/bash_paths
dis tools add-rc-path \
  --name 'sample hello path' \
  --content 'export PATH="$HOME/.local/share/sample-hello/bin:$PATH"'

# Register shell init code — appears in ~/rc/configs-generated/bash_init
dis tools add-rc-init \
  --name 'sample hello init' \
  --content 'echo "hello from dis sample installer!"'

# Wire a line into ~/.bashrc — uses add-home-rc
dis tools add-home-rc \
  --name 'sample hello home-rc' \
  --content '# added by sample/hello installer'

echo "==> sample/hello done"
echo "    Check the results:"
echo "      cat ~/rc/configs-generated/bash_paths"
echo "      cat ~/rc/configs-generated/bash_init"
echo "      cat ~/.bashrc"
