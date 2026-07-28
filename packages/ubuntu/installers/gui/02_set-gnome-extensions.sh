### -- Manifest
### provides: gui/gnome-extensions
### depends_on: [gui/gnome-tweak-tool]
### distro: [ubuntu]
### -- End
# Quick guide on gnome extensions
#
# You can list installed extensions with:
# gext list
#
# Schema extensions allow to configure the extension config programmatically.
# To list schema extensions:
# gsettings list-schemas | grep extens
# 
# Then you can find which config keys it supports with:
# gsettings list-keys org.gnome.shell.extensions.switcher
#
# or
# gsettings list-recursively org.gnome.shell.extensions.switcher


if [[ "$XDG_SESSION_TYPE" == "tty" ]]; then
    echo "GUI install not available on tty session type"
    exit 0
fi

sudo apt install -y gnome-shell-extension-manager pipx

# Turn off default Ubuntu extensions
# need to disable otherwise the shortcuts conflicts with apps
gnome-extensions disable ubuntu-dock@ubuntu.com

# Install the gnome-extension-cli (gext) [1]
#
# [1] https://github.com/essembeh/gnome-extensions-cli
pipx install gnome-extensions-cli --system-site-packages
# Note that to start using gnome-extensions-cli (gext), $HOME/.local/bin needs to be in PATH.
bashrc_path_add '$HOME/.local/bin path' 'export PATH="$HOME/.local/bin:$PATH"'
# To start using getx from the remaining of the commands, we are going to explicitly add it to the path.
export PATH=$HOME/.local/bin:$PATH


# Pause to assure user is ready to accept confirmations
#
# In theory it should be possible to use gext --filesystem option which  allows installing extensions without any Gnome 
# session running (over ssh for example or headless). Unfortunately that uses the non native way to install 
# Gnome extensions and does not always work. So for this works only if:
# - this is running from a gnome session
# - you are ready to ack the prompt that the Gnome shows when installing extensions.
read -p "To install Gnome extensions, you need to accept some confirmations. Are you ready? " -n 1 -r
if [[ ! $REPLY =~ ^[Yy]$ ]]
then
    # handle exits from shell or function but don't exit interactive shell
    [[ "$0" = "$BASH_SOURCE" ]] && exit 1 || return 1 
fi

gext install switcher@landau.fi                                      # Switch windows or launch applications quickly by typing, similar to Alfred/Albert.
# gext install tactile@lundal.io                                     # Tile windows on a custom grid using your keyboard.
gext install tilingshell@ferrarodomenico.com                         # Extend Gnome Shell with advanced tiling window management (https://extensions.gnome.org/extension/7065/tiling-shell/).
gext install clipboard-indicator@tudmotu.com                         # Clipboard manager.
gext install ddterm@amezin.github.com                                # Drop down terminal extension for GNOME Shell. With tabs. Works on Wayland natively.

# Compile gsettings schemas in order to be able to set extension configs
# sudo cp ~/.local/share/gnome-shell/extensions/tactile@lundal.io/schemas/org.gnome.shell.extensions.tactile.gschema.xml /usr/share/glib-2.0/schemas/
sudo cp ~/.local/share/gnome-shell/extensions/switcher@landau.fi/schemas/org.gnome.shell.extensions.switcher.gschema.xml /usr/share/glib-2.0/schemas/
sudo cp ~/.local/share/gnome-shell/extensions/ddterm@amezin.github.com/schemas/org.gnome.shell.extensions.ddterm.gschema.xml /usr/share/glib-2.0/schemas
sudo cp ~/.local/share/gnome-shell/extensions/tilingshell\@ferrarodomenico.com/schemas/org.gnome.shell.extensions.tilingshell.gschema.xml /usr/share/glib-2.0/schemas
sudo glib-compile-schemas /usr/share/glib-2.0/schemas/

# Configure Switcher
gsettings set org.gnome.shell.extensions.switcher show-switcher "['<Super>home']"
gsettings set org.gnome.shell.extensions.switcher max-width-percentage 60
gsettings set org.gnome.shell.extensions.switcher font-size 20
gsettings set org.gnome.shell.extensions.switcher icon-size 32

# Configure tilingshell
gsettings set org.gnome.shell.extensions.tilingshell move-window-down "['<Super>Down']"
gsettings set org.gnome.shell.extensions.tilingshell move-window-left "['<Super>Left']"
gsettings set org.gnome.shell.extensions.tilingshell move-window-right "['<Super>Right']"
gsettings set org.gnome.shell.extensions.tilingshell move-window-up "['<Super>Up']"
gsettings set org.gnome.shell.extensions.tilingshell span-window-all-tiles "['<Super>slash']"
gsettings set org.gnome.shell.extensions.tilingshell enable-autotiling true
gsettings set org.gnome.shell.extensions.tilingshell inner-gaps 10
gsettings set org.gnome.shell.extensions.tilingshell outer-gaps 14
# Disable automaximize which maximizes the window when the window initial size covers
# almost all the screen (interferes with a 1 tile layout).
gsettings set org.gnome.mutter auto-maximize false

# Configure ddterm
#
# Get the ddterm GSettings schema.
# New versions use "org.gnome.shell.extensions.ddterm".
# Old versions use "com.github.amezin.ddterm".
ddterm_schema() {
    if gsettings list-schemas | grep -qx 'org.gnome.shell.extensions.ddterm'; then
        printf '%s\n' 'org.gnome.shell.extensions.ddterm'
    else
        printf '%s\n' 'com.github.amezin.ddterm'
    fi
}

SCHEMA=$(ddterm_schema)

gsettings set $SCHEMA allow-hyperlink true
gsettings set $SCHEMA audible-bell true
gsettings set $SCHEMA background-color '#181818'
gsettings set $SCHEMA background-opacity 0.90000000000000000
gsettings set $SCHEMA backspace-binding 'ascii-delete'
gsettings set $SCHEMA bold-color '#000000'
gsettings set $SCHEMA bold-color-same-as-fg true
gsettings set $SCHEMA bold-is-bright false
gsettings set $SCHEMA cjk-utf8-ambiguous-width 'narrow'
gsettings set $SCHEMA command 'user-shell'
gsettings set $SCHEMA cursor-background-color '#000000'
gsettings set $SCHEMA cursor-blink-mode 'system'
gsettings set $SCHEMA cursor-colors-set false
gsettings set $SCHEMA cursor-foreground-color '#ffffff'
gsettings set $SCHEMA cursor-shape 'block'
gsettings set $SCHEMA custom-command ''
gsettings set $SCHEMA custom-font 'Monospace Regular 10'
gsettings set $SCHEMA ddterm-toggle-hotkey "['F12']"
gsettings set $SCHEMA delete-binding 'delete-sequence'
gsettings set $SCHEMA detect-urls true
gsettings set $SCHEMA detect-urls-as-is true
gsettings set $SCHEMA detect-urls-email true
gsettings set $SCHEMA detect-urls-file true
gsettings set $SCHEMA detect-urls-http true
gsettings set $SCHEMA detect-urls-news-man true
gsettings set $SCHEMA detect-urls-voip true
gsettings set $SCHEMA force-x11-gdk-backend false
gsettings set $SCHEMA foreground-color '#171421'
gsettings set $SCHEMA hide-animation 'ease-in-quad'
gsettings set $SCHEMA hide-animation-duration 0.15000000000000000
gsettings set $SCHEMA hide-when-focus-lost true
gsettings set $SCHEMA hide-window-on-esc false
gsettings set $SCHEMA highlight-background-color '#000000'
gsettings set $SCHEMA highlight-colors-set false
gsettings set $SCHEMA highlight-foreground-color '#ffffff'
gsettings set $SCHEMA new-tab-button true
gsettings set $SCHEMA new-tab-front-button false
gsettings set $SCHEMA notebook-border true
gsettings set $SCHEMA override-window-animation true
gsettings set $SCHEMA palette "['#171421', '#c01c28', '#26a269', '#a2734c', '#12488b', '#a347ba', '#2aa1b3', '#d0cfcc', '#5e5c64', '#f66151', '#33da7a', '#e9ad0c', '#2a7bde', '#c061cb', '#33c7de', '#ffffff']"
gsettings set $SCHEMA panel-icon-type 'toggle-and-menu-button'
gsettings set $SCHEMA pointer-autohide false
gsettings set $SCHEMA preserve-working-directory true
gsettings set $SCHEMA scroll-on-keystroke true
gsettings set $SCHEMA scroll-on-output false
gsettings set $SCHEMA scrollback-lines 10000
gsettings set $SCHEMA scrollback-unlimited false
gsettings set $SCHEMA shortcut-find "['<Ctrl><Shift>F']"
gsettings set $SCHEMA shortcut-find-next "['<Ctrl><Shift>G']"
gsettings set $SCHEMA shortcut-find-prev "['<Ctrl><Shift>H']"
gsettings set $SCHEMA shortcut-font-scale-decrease "['<Ctrl>minus']"
gsettings set $SCHEMA shortcut-font-scale-increase "['<Ctrl>plus']"
gsettings set $SCHEMA shortcut-font-scale-reset "['<Ctrl>0']"
gsettings set $SCHEMA shortcut-move-tab-next "['<Ctrl><Shift>Page_Down']"
gsettings set $SCHEMA shortcut-move-tab-prev "['<Ctrl><Shift>Page_Up']"
gsettings set $SCHEMA shortcut-next-tab "['<Ctrl>Page_Down']"
gsettings set $SCHEMA shortcut-page-close "['<Ctrl><Shift>q']"
gsettings set $SCHEMA shortcut-prev-tab "['<Ctrl>Page_Up']"
gsettings set $SCHEMA shortcut-switch-to-tab-1 "['<Alt>1']"
gsettings set $SCHEMA shortcut-switch-to-tab-10 "['<Alt>0']"
gsettings set $SCHEMA shortcut-switch-to-tab-2 "['<Alt>2']"
gsettings set $SCHEMA shortcut-switch-to-tab-3 "['<Alt>3']"
gsettings set $SCHEMA shortcut-switch-to-tab-4 "['<Alt>4']"
gsettings set $SCHEMA shortcut-switch-to-tab-5 "['<Alt>5']"
gsettings set $SCHEMA shortcut-switch-to-tab-6 "['<Alt>6']"
gsettings set $SCHEMA shortcut-switch-to-tab-7 "['<Alt>7']"
gsettings set $SCHEMA shortcut-switch-to-tab-8 "['<Alt>8']"
gsettings set $SCHEMA shortcut-switch-to-tab-9 "['<Alt>9']"
gsettings set $SCHEMA shortcut-terminal-copy "['<Ctrl><Shift>c']"
gsettings set $SCHEMA shortcut-terminal-paste "['<Ctrl><Shift>v']"
gsettings set $SCHEMA shortcut-toggle-maximize "['F11']"
gsettings set $SCHEMA shortcut-win-new-tab "['<Ctrl><Shift>n']"
gsettings set $SCHEMA shortcut-window-size-dec "['<Ctrl>Up']"
gsettings set $SCHEMA shortcut-window-size-inc "['<Ctrl>Down']"
gsettings set $SCHEMA shortcuts-enabled true
gsettings set $SCHEMA show-animation 'linear'
gsettings set $SCHEMA show-animation-duration 0.14999999999999999
gsettings set $SCHEMA show-scrollbar true
gsettings set $SCHEMA tab-close-buttons true
gsettings set $SCHEMA tab-expand true
gsettings set $SCHEMA tab-label-ellipsize-mode 'none'
gsettings set $SCHEMA tab-label-width 0.10000000000000001
gsettings set $SCHEMA tab-policy 'never'
gsettings set $SCHEMA tab-position 'bottom'
gsettings set $SCHEMA tab-show-shortcuts true
gsettings set $SCHEMA tab-switcher-popup true
gsettings set $SCHEMA text-blink-mode 'always'
gsettings set $SCHEMA theme-variant 'system'
gsettings set $SCHEMA transparent-background true
gsettings set $SCHEMA use-system-font true
gsettings set $SCHEMA use-theme-colors true
gsettings set $SCHEMA window-above true
gsettings set $SCHEMA window-maximize true
gsettings set $SCHEMA window-monitor 'current'
gsettings set $SCHEMA window-monitor-connector ''
gsettings set $SCHEMA window-position 'top'
gsettings set $SCHEMA window-resizable false
gsettings set $SCHEMA window-size 1.0
gsettings set $SCHEMA window-skip-taskbar true
gsettings set $SCHEMA window-stick true

