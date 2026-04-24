#!/usr/bin/env bash
### -- Manifest
### provides: common/ssh-agent
### depends_on: []
### distro: [ubuntu]
### -- End

# --------------------------------------------------------------------
# This script configures a persistent systemd user service for ssh-agent
# on a Linux system with systemd (e.g., Ubuntu 22.04+).
#
# What it does:
#   1. Creates ~/.config/systemd/user/ssh-agent.service to run ssh-agent
#      as a persistent socket-activated systemd user service.
#   2. Creates ~/.config/environment.d/ssh_auth_socket.conf — NOTE: this
#      only affects systemd-started user services, NOT interactive shells.
#      See "Why bashrc?" below.
#   3. Enables and starts the ssh-agent systemd user service.
#   4. Adds SSH_AUTH_SOCK export to ~/rc/configs-generated/bash_init via
#      bashrc_init_add, so every shell (SSH or local) sees the agent.
#
# --------------------------------------------------------------------
# Why bashrc and not just environment.d?
# --------------------------------------------------------------------
#
# environment.d files are read by systemd-environment-d-generator and
# injected into the systemd USER SESSION environment. That environment
# is only visible to processes started by systemd --user (e.g., other
# user services), NOT to interactive shells.
#
# XDG_RUNTIME_DIR IS set in SSH sessions because pam_systemd injects it
# directly into the PAM environment. But SSH_AUTH_SOCK is NOT — it must
# be exported explicitly in the shell's startup files.
#
# This is orthogonal to SSH: the same problem occurs in a local terminal
# on a headless server. The environment.d mechanism was designed for
# systemd-managed graphical sessions (GNOME etc.), where the desktop
# session manager bridges the systemd env into the shell. On headless
# servers and SSH sessions there is no such bridge.
#
# Session type → who sets SSH_AUTH_SOCK:
#
#   Graphical login (GDM/GNOME)  → GNOME session manager / environment.d ✅
#   Local terminal (after GUI)   → inherited from graphical session      ✅
#   ssh user@host                → nobody, unless ~/.bashrc sets it      ❌→✅
#   su - user (local)            → nobody, unless ~/.bashrc sets it      ❌→✅
#
# Flow diagram:
#
#   ssh user@host
#       │
#       ├── pam_systemd → sets XDG_RUNTIME_DIR in PAM env ──→ reaches shell ✅
#       │                → starts systemd --user
#       │                    └── reads environment.d ──────→ systemd env only ❌
#       │
#       └── shell starts
#               └── sources ~/.bashrc
#                       └── export SSH_AUTH_SOCK=... ──────→ reaches shell ✅
#
# References:
#   environment.d(5):
#     https://www.freedesktop.org/software/systemd/man/latest/environment.d.html
#   systemd.environment-generator(7):
#     https://www.freedesktop.org/software/systemd/man/latest/systemd.environment-generator.html
#
# After running:
#   - Open a new shell (reconnect SSH or open a new terminal).
#   - Add your key once: ssh-add ~/.ssh/<your_key>
#   - From then on, the agent persists across logins and shells.
# --------------------------------------------------------------------
source $DIS_BINDING
set -euo pipefail

SERVICE_DIR="$HOME/.config/systemd/user"
SERVICE_FILE="$SERVICE_DIR/ssh-agent.service"
ENV_DIR="$HOME/.config/environment.d"
ENV_FILE="$ENV_DIR/ssh_auth_socket.conf"

if [ -f $ENV_FILE ]; then
    echo "ssh agent already set up"
    exit 0
fi

echo "📂 Creating systemd user service directory..."
mkdir -p "$SERVICE_DIR"

echo "📝 Writing ssh-agent.service..."
cat > "$SERVICE_FILE" <<'EOF'
[Unit]
Description=SSH key agent

[Service]
Type=simple
Environment=SSH_AUTH_SOCK=%t/ssh-agent.socket
ExecStart=/usr/bin/ssh-agent -D -a $SSH_AUTH_SOCK

[Install]
WantedBy=default.target
EOF

echo "🔄 Reloading systemd user units..."
systemctl --user daemon-reload

echo "✅ Enabling and starting ssh-agent service..."
systemctl --user enable ssh-agent
systemctl --user start ssh-agent

echo "📂 Creating environment.d directory..."
mkdir -p "$ENV_DIR"

echo "📝 Writing SSH_AUTH_SOCK to $ENV_FILE"
cat > "$ENV_FILE" <<'EOF'
SSH_AUTH_SOCK=${XDG_RUNTIME_DIR}/ssh-agent.socket
EOF

echo "🔄 Reloading systemd user environment..."
systemctl --user import-environment SSH_AUTH_SOCK || true
bashrc_init_add "SSH agent socket" \
  'export SSH_AUTH_SOCK="${XDG_RUNTIME_DIR}/ssh-agent.socket"'

echo "🎉 Done!"
echo "➡️  Log out and back in, or run:"
echo "   export SSH_AUTH_SOCK=\$XDG_RUNTIME_DIR/ssh-agent.socket"
echo "to make sure new shells see the agent."
echo "Then you are ready to add you ssh key:"
echo "   ssh-add ~/.ssh/<your_key>"
