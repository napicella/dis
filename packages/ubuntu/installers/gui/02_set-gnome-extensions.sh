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

source $DIS_BINDING

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
gext install tactile@lundal.io                                       # Tile windows on a custom grid using your keyboard.
gext install clipboard-indicator@tudmotu.com                         # Clipboard manager.
gext install ddterm@amezin.github.com                                # Drop down terminal extension for GNOME Shell. With tabs. Works on Wayland natively.
gext install tilingshell@ferrarodomenico.com                         # Extend Gnome Shell with advanced tiling window management (https://extensions.gnome.org/extension/7065/tiling-shell/). 

# for some reason, I need to make the extension binary executable, at least for ddterm.
# chmod +x $HOME/.local/share/gnome-shell/extensions/ddterm@amezin.github.com/bin/com.github.amezin.ddterm

# Compile gsettings schemas in order to be able to set extension configs
sudo cp ~/.local/share/gnome-shell/extensions/tactile@lundal.io/schemas/org.gnome.shell.extensions.tactile.gschema.xml /usr/share/glib-2.0/schemas/
sudo cp ~/.local/share/gnome-shell/extensions/switcher@landau.fi/schemas/org.gnome.shell.extensions.switcher.gschema.xml /usr/share/glib-2.0/schemas/
sudo cp ~/.local/share/gnome-shell/extensions/ddterm@amezin.github.com/schemas/org.gnome.shell.extensions.ddterm.gschema.xml /usr/share/glib-2.0/schemas
sudo glib-compile-schemas /usr/share/glib-2.0/schemas/

# Configure Tactile
gsettings set org.gnome.shell.extensions.tactile col-0 1
gsettings set org.gnome.shell.extensions.tactile col-1 2
gsettings set org.gnome.shell.extensions.tactile col-2 1
gsettings set org.gnome.shell.extensions.tactile col-3 0
gsettings set org.gnome.shell.extensions.tactile row-0 1
gsettings set org.gnome.shell.extensions.tactile row-1 1
gsettings set org.gnome.shell.extensions.tactile gap-size 32

# Configure Switcher
gsettings set org.gnome.shell.extensions.switcher show-switcher "['<Super>home']"
gsettings set org.gnome.shell.extensions.switcher max-width-percentage 60
gsettings set org.gnome.shell.extensions.switcher font-size 20
gsettings set org.gnome.shell.extensions.switcher icon-size 32

# Configure ddterm
gsettings set org.gnome.shell.extensions.ddterm allow-hyperlink true
gsettings set org.gnome.shell.extensions.ddterm audible-bell true
gsettings set org.gnome.shell.extensions.ddterm background-color '#ffffff'
gsettings set org.gnome.shell.extensions.ddterm background-opacity 0.90000000000000000
gsettings set org.gnome.shell.extensions.ddterm backspace-binding 'ascii-delete'
gsettings set org.gnome.shell.extensions.ddterm bold-color '#000000'
gsettings set org.gnome.shell.extensions.ddterm bold-color-same-as-fg true
gsettings set org.gnome.shell.extensions.ddterm bold-is-bright false
gsettings set org.gnome.shell.extensions.ddterm cjk-utf8-ambiguous-width 'narrow'
gsettings set org.gnome.shell.extensions.ddterm command 'user-shell'
gsettings set org.gnome.shell.extensions.ddterm cursor-background-color '#000000'
gsettings set org.gnome.shell.extensions.ddterm cursor-blink-mode 'system'
gsettings set org.gnome.shell.extensions.ddterm cursor-colors-set false
gsettings set org.gnome.shell.extensions.ddterm cursor-foreground-color '#ffffff'
gsettings set org.gnome.shell.extensions.ddterm cursor-shape 'block'
gsettings set org.gnome.shell.extensions.ddterm custom-command ''
gsettings set org.gnome.shell.extensions.ddterm custom-font 'Monospace Regular 10'
gsettings set org.gnome.shell.extensions.ddterm ddterm-toggle-hotkey "['F12']"
gsettings set org.gnome.shell.extensions.ddterm delete-binding 'delete-sequence'
gsettings set org.gnome.shell.extensions.ddterm detect-urls true
gsettings set org.gnome.shell.extensions.ddterm detect-urls-as-is true
gsettings set org.gnome.shell.extensions.ddterm detect-urls-email true
gsettings set org.gnome.shell.extensions.ddterm detect-urls-file true
gsettings set org.gnome.shell.extensions.ddterm detect-urls-http true
gsettings set org.gnome.shell.extensions.ddterm detect-urls-news-man true
gsettings set org.gnome.shell.extensions.ddterm detect-urls-voip true
gsettings set org.gnome.shell.extensions.ddterm force-x11-gdk-backend false
gsettings set org.gnome.shell.extensions.ddterm foreground-color '#171421'
gsettings set org.gnome.shell.extensions.ddterm hide-animation 'ease-in-quad'
gsettings set org.gnome.shell.extensions.ddterm hide-animation-duration 0.15000000000000000
gsettings set org.gnome.shell.extensions.ddterm hide-when-focus-lost true
gsettings set org.gnome.shell.extensions.ddterm hide-window-on-esc false
gsettings set org.gnome.shell.extensions.ddterm highlight-background-color '#000000'
gsettings set org.gnome.shell.extensions.ddterm highlight-colors-set false
gsettings set org.gnome.shell.extensions.ddterm highlight-foreground-color '#ffffff'
gsettings set org.gnome.shell.extensions.ddterm new-tab-button true
gsettings set org.gnome.shell.extensions.ddterm new-tab-front-button false
gsettings set org.gnome.shell.extensions.ddterm notebook-border true
gsettings set org.gnome.shell.extensions.ddterm override-window-animation true
gsettings set org.gnome.shell.extensions.ddterm palette "['#171421', '#c01c28', '#26a269', '#a2734c', '#12488b', '#a347ba', '#2aa1b3', '#d0cfcc', '#5e5c64', '#f66151', '#33da7a', '#e9ad0c', '#2a7bde', '#c061cb', '#33c7de', '#ffffff']"
gsettings set org.gnome.shell.extensions.ddterm panel-icon-type 'toggle-and-menu-button'
gsettings set org.gnome.shell.extensions.ddterm pointer-autohide false
gsettings set org.gnome.shell.extensions.ddterm preserve-working-directory true
gsettings set org.gnome.shell.extensions.ddterm scroll-on-keystroke true
gsettings set org.gnome.shell.extensions.ddterm scroll-on-output false
gsettings set org.gnome.shell.extensions.ddterm scrollback-lines 10000
gsettings set org.gnome.shell.extensions.ddterm scrollback-unlimited false
#gsettings set org.gnome.shell.extensions.ddterm shortcut-background-opacity-dec @as "[]"
#gsettings set org.gnome.shell.extensions.ddterm shortcut-background-opacity-inc @as "[]"
gsettings set org.gnome.shell.extensions.ddterm shortcut-find "['<Ctrl><Shift>F']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-find-next "['<Ctrl><Shift>G']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-find-prev "['<Ctrl><Shift>H']"
# gsettings set org.gnome.shell.extensions.ddterm shortcut-focus-other-pane @as []
gsettings set org.gnome.shell.extensions.ddterm shortcut-font-scale-decrease "['<Ctrl>minus']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-font-scale-increase "['<Ctrl>plus']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-font-scale-reset "['<Ctrl>0']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-move-tab-next "['<Ctrl><Shift>Page_Down']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-move-tab-prev "['<Ctrl><Shift>Page_Up']"
#gsettings set org.gnome.shell.extensions.ddterm shortcut-move-tab-to-other-pane @as []
gsettings set org.gnome.shell.extensions.ddterm shortcut-next-tab "['<Ctrl>Page_Down']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-page-close "['<Ctrl><Shift>q']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-prev-tab "['<Ctrl>Page_Up']"
# gsettings set org.gnome.shell.extensions.ddterm shortcut-reset-tab-title @as []
# gsettings set org.gnome.shell.extensions.ddterm shortcut-set-custom-tab-title @as []
# gsettings set org.gnome.shell.extensions.ddterm shortcut-split-horizontal @as []
# gsettings set org.gnome.shell.extensions.ddterm shortcut-split-position-dec @as []
# gsettings set org.gnome.shell.extensions.ddterm shortcut-split-position-inc @as []
# gsettings set org.gnome.shell.extensions.ddterm shortcut-split-vertical @as []
gsettings set org.gnome.shell.extensions.ddterm shortcut-switch-to-tab-1 "['<Alt>1']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-switch-to-tab-10 "['<Alt>0']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-switch-to-tab-2 "['<Alt>2']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-switch-to-tab-3 "['<Alt>3']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-switch-to-tab-4 "['<Alt>4']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-switch-to-tab-5 "['<Alt>5']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-switch-to-tab-6 "['<Alt>6']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-switch-to-tab-7 "['<Alt>7']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-switch-to-tab-8 "['<Alt>8']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-switch-to-tab-9 "['<Alt>9']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-terminal-copy "['<Ctrl><Shift>c']"
# gsettings set org.gnome.shell.extensions.ddterm shortcut-terminal-copy-html @as []
gsettings set org.gnome.shell.extensions.ddterm shortcut-terminal-paste "['<Ctrl><Shift>v']"
# gsettings set org.gnome.shell.extensions.ddterm shortcut-terminal-reset @as []
# gsettings set org.gnome.shell.extensions.ddterm shortcut-terminal-reset-and-clear @as []
# gsettings set org.gnome.shell.extensions.ddterm shortcut-terminal-select-all @as []
gsettings set org.gnome.shell.extensions.ddterm shortcut-toggle-maximize "['F11']"
# gsettings set org.gnome.shell.extensions.ddterm shortcut-toggle-transparent-background @as []
gsettings set org.gnome.shell.extensions.ddterm shortcut-win-new-tab "['<Ctrl><Shift>n']"
# gsettings set org.gnome.shell.extensions.ddterm shortcut-win-new-tab-after-current @as []
# gsettings set org.gnome.shell.extensions.ddterm shortcut-win-new-tab-before-current @as []
# gsettings set org.gnome.shell.extensions.ddterm shortcut-win-new-tab-front @as []
# gsettings set org.gnome.shell.extensions.ddterm shortcut-window-hide @as []
gsettings set org.gnome.shell.extensions.ddterm shortcut-window-size-dec "['<Ctrl>Up']"
gsettings set org.gnome.shell.extensions.ddterm shortcut-window-size-inc "['<Ctrl>Down']"
gsettings set org.gnome.shell.extensions.ddterm shortcuts-enabled true
gsettings set org.gnome.shell.extensions.ddterm show-animation 'linear'
gsettings set org.gnome.shell.extensions.ddterm show-animation-duration 0.14999999999999999
gsettings set org.gnome.shell.extensions.ddterm show-scrollbar true
gsettings set org.gnome.shell.extensions.ddterm tab-close-buttons true
gsettings set org.gnome.shell.extensions.ddterm tab-expand true
gsettings set org.gnome.shell.extensions.ddterm tab-label-ellipsize-mode 'none'
gsettings set org.gnome.shell.extensions.ddterm tab-label-width 0.10000000000000001
gsettings set org.gnome.shell.extensions.ddterm tab-policy 'never'
gsettings set org.gnome.shell.extensions.ddterm tab-position 'bottom'
gsettings set org.gnome.shell.extensions.ddterm tab-show-shortcuts true
gsettings set org.gnome.shell.extensions.ddterm tab-switcher-popup true
gsettings set org.gnome.shell.extensions.ddterm text-blink-mode 'always'
gsettings set org.gnome.shell.extensions.ddterm theme-variant 'system'
gsettings set org.gnome.shell.extensions.ddterm transparent-background true
gsettings set org.gnome.shell.extensions.ddterm use-system-font true
gsettings set org.gnome.shell.extensions.ddterm use-theme-colors true
gsettings set org.gnome.shell.extensions.ddterm window-above true
gsettings set org.gnome.shell.extensions.ddterm window-maximize true
gsettings set org.gnome.shell.extensions.ddterm window-monitor 'current'
gsettings set org.gnome.shell.extensions.ddterm window-monitor-connector ''
gsettings set org.gnome.shell.extensions.ddterm window-position 'top'
gsettings set org.gnome.shell.extensions.ddterm window-resizable false
gsettings set org.gnome.shell.extensions.ddterm window-size 1.0
gsettings set org.gnome.shell.extensions.ddterm window-skip-taskbar true
gsettings set org.gnome.shell.extensions.ddterm window-stick true

