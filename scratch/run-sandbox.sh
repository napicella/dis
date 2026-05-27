#!/usr/bin/env bash
# scratch/run-sandbox.sh
#
# Spin up an interactive Ubuntu container for manually experimenting with dis.
# The dis binary is built from local source. The scratch/ directory and the
# dis/packages directory are bind-mounted so edits on the host are instantly
# visible inside the container.
#
# Usage:
#   cd /path/to/dis
#   ./scratch/run-sandbox.sh
#
# Inside the container run:
#   dis install --distro /sandbox/distro.yml
#
# To clean up: just exit the shell — the container is --rm'd automatically.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# ── Build the binary ──────────────────────────────────────────────────────────
echo "==> Building dis binary..."
cd "$DIS_ROOT"
go build -o "$DIS_ROOT/scratch/dis" .
echo "    done: $DIS_ROOT/scratch/dis"

# ── Write the sandbox distro YAML ─────────────────────────────────────────────
DISTRO_FILE="$DIS_ROOT/scratch/distro.yml"
cat > "$DISTRO_FILE" <<'YAML'
os: ubuntu
sources:
  - /sandbox/packages   # scratch/packages/ — sample installers
  - /dis/packages       # dis built-in packages

packages:
  - common/os-libs
  - sample/hello
YAML
echo "==> Distro file written: $DISTRO_FILE"

# ── Start the container ───────────────────────────────────────────────────────
echo "==> Starting sandbox container..."
echo "    scratch/   → /sandbox       (bind mount, read-write)"
echo "    dis/packages → /dis/packages (bind mount, read-only)"
echo ""
echo "Inside the container run:"
echo "  dis install --distro /sandbox/distro.yml"
echo ""

docker run --rm -it \
  --name dis-sandbox \
  -v "$DIS_ROOT/packages:/dis/packages:ro" \
  -v "$DIS_ROOT/scratch:/sandbox" \
  ubuntu:22.04 \
  bash -c '
    set -e
    # Install sudo and create dev user
    apt-get update -qq
    apt-get install -y -qq sudo
    useradd -m -s /bin/bash dev 2>/dev/null || true
    echo "dev ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers
    # Install the dis binary
    cp /sandbox/dis /usr/local/bin/dis
    chmod +x /usr/local/bin/dis
    echo "dis is ready at /usr/local/bin/dis"
    echo "Run: dis install --distro /sandbox/distro.yml"
    # Drop into an interactive shell as dev
    exec su - dev
  '
