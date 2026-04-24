### -- Manifest
### provides: common/gnome-extensions
### depends_on: []
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
gum confirm "To install Gnome extensions, you need to accept some confirmations. Are you ready?"

gext install switcher@landau.fi                                      # Switch windows or launch applications quickly by typing, similar to Alfred/Albert.
gext install tactile@lundal.io                                       # Tile windows on a custom grid using your keyboard.
gext install clipboard-indicator@tudmotu.com                         # Clipboard manager.
gext install ddterm@amezin.github.com                                # Drop down terminal extension for GNOME Shell. With tabs. Works on Wayland natively.
gext install Current_screen_only_for_Alternate_Tab@bourcereau.fr     # Limits the windows shown on the switcher to those of the current monitor (https://extensions.gnome.org/extension/1437/current-screen-only-for-alternate-tab/)

# for some reason, I need to make the extension binary executable, at least for ddterm.
chmod +x $HOME/.local/share/gnome-shell/extensions/ddterm@amezin.github.com/bin/com.github.amezin.ddterm

# Compile gsettings schemas in order to be able to set extension configs
sudo cp ~/.local/share/gnome-shell/extensions/tactile@lundal.io/schemas/org.gnome.shell.extensions.tactile.gschema.xml /usr/share/glib-2.0/schemas/
sudo cp ~/.local/share/gnome-shell/extensions/switcher@landau.fi/schemas/org.gnome.shell.extensions.switcher.gschema.xml /usr/share/glib-2.0/schemas/
sudo cp ~/.local/share/gnome-shell/extensions/ddterm@amezin.github.com/schemas/com.github.amezin.ddterm.gschema.xml /usr/share/glib-2.0/schemas
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
gsettings set org.gnome.shell.extensions.switcher show-switcher "['<Super>q']"
gsettings set org.gnome.shell.extensions.switcher max-width-percentage 60
gsettings set org.gnome.shell.extensions.switcher font-size 20
gsettings set org.gnome.shell.extensions.switcher icon-size 32

# Configure ddterm
gsettings set com.github.amezin.ddterm allow-hyperlink true
gsettings set com.github.amezin.ddterm audible-bell true
gsettings set com.github.amezin.ddterm background-color '#ffffff'
gsettings set com.github.amezin.ddterm background-opacity 0.90000000000000000
gsettings set com.github.amezin.ddterm backspace-binding 'ascii-delete'
gsettings set com.github.amezin.ddterm bold-color '#000000'
gsettings set com.github.amezin.ddterm bold-color-same-as-fg true
gsettings set com.github.amezin.ddterm bold-is-bright false
gsettings set com.github.amezin.ddterm cjk-utf8-ambiguous-width 'narrow'
gsettings set com.github.amezin.ddterm command 'user-shell'
gsettings set com.github.amezin.ddterm cursor-background-color '#000000'
gsettings set com.github.amezin.ddterm cursor-blink-mode 'system'
gsettings set com.github.amezin.ddterm cursor-colors-set false
gsettings set com.github.amezin.ddterm cursor-foreground-color '#ffffff'
gsettings set com.github.amezin.ddterm cursor-shape 'block'
gsettings set com.github.amezin.ddterm custom-command ''
gsettings set com.github.amezin.ddterm custom-font 'Monospace Regular 10'
gsettings set com.github.amezin.ddterm ddterm-toggle-hotkey "['F12']"
gsettings set com.github.amezin.ddterm delete-binding 'delete-sequence'
gsettings set com.github.amezin.ddterm detect-urls true
gsettings set com.github.amezin.ddterm detect-urls-as-is true
gsettings set com.github.amezin.ddterm detect-urls-email true
gsettings set com.github.amezin.ddterm detect-urls-file true
gsettings set com.github.amezin.ddterm detect-urls-http true
gsettings set com.github.amezin.ddterm detect-urls-news-man true
gsettings set com.github.amezin.ddterm detect-urls-voip true
gsettings set com.github.amezin.ddterm force-x11-gdk-backend false
gsettings set com.github.amezin.ddterm foreground-color '#171421'
gsettings set com.github.amezin.ddterm hide-animation 'ease-in-quad'
gsettings set com.github.amezin.ddterm hide-animation-duration 0.15000000000000000
gsettings set com.github.amezin.ddterm hide-when-focus-lost true
gsettings set com.github.amezin.ddterm hide-window-on-esc false
gsettings set com.github.amezin.ddterm highlight-background-color '#000000'
gsettings set com.github.amezin.ddterm highlight-colors-set false
gsettings set com.github.amezin.ddterm highlight-foreground-color '#ffffff'
gsettings set com.github.amezin.ddterm new-tab-button true
gsettings set com.github.amezin.ddterm new-tab-front-button false
gsettings set com.github.amezin.ddterm notebook-border true
gsettings set com.github.amezin.ddterm override-window-animation true
gsettings set com.github.amezin.ddterm palette "['#171421', '#c01c28', '#26a269', '#a2734c', '#12488b', '#a347ba', '#2aa1b3', '#d0cfcc', '#5e5c64', '#f66151', '#33da7a', '#e9ad0c', '#2a7bde', '#c061cb', '#33c7de', '#ffffff']"
gsettings set com.github.amezin.ddterm panel-icon-type 'toggle-and-menu-button'
gsettings set com.github.amezin.ddterm pointer-autohide false
gsettings set com.github.amezin.ddterm preserve-working-directory true
gsettings set com.github.amezin.ddterm scroll-on-keystroke true
gsettings set com.github.amezin.ddterm scroll-on-output false
gsettings set com.github.amezin.ddterm scrollback-lines 10000
gsettings set com.github.amezin.ddterm scrollback-unlimited false
#gsettings set com.github.amezin.ddterm shortcut-background-opacity-dec @as "[]"
#gsettings set com.github.amezin.ddterm shortcut-background-opacity-inc @as "[]"
gsettings set com.github.amezin.ddterm shortcut-find "['<Ctrl><Shift>F']"
gsettings set com.github.amezin.ddterm shortcut-find-next "['<Ctrl><Shift>G']"
gsettings set com.github.amezin.ddterm shortcut-find-prev "['<Ctrl><Shift>H']"
# gsettings set com.github.amezin.ddterm shortcut-focus-other-pane @as []
gsettings set com.github.amezin.ddterm shortcut-font-scale-decrease "['<Ctrl>minus']"
gsettings set com.github.amezin.ddterm shortcut-font-scale-increase "['<Ctrl>plus']"
gsettings set com.github.amezin.ddterm shortcut-font-scale-reset "['<Ctrl>0']"
gsettings set com.github.amezin.ddterm shortcut-move-tab-next "['<Ctrl><Shift>Page_Down']"
gsettings set com.github.amezin.ddterm shortcut-move-tab-prev "['<Ctrl><Shift>Page_Up']"
#gsettings set com.github.amezin.ddterm shortcut-move-tab-to-other-pane @as []
gsettings set com.github.amezin.ddterm shortcut-next-tab "['<Ctrl>Page_Down']"
gsettings set com.github.amezin.ddterm shortcut-page-close "['<Ctrl><Shift>q']"
gsettings set com.github.amezin.ddterm shortcut-prev-tab "['<Ctrl>Page_Up']"
# gsettings set com.github.amezin.ddterm shortcut-reset-tab-title @as []
# gsettings set com.github.amezin.ddterm shortcut-set-custom-tab-title @as []
# gsettings set com.github.amezin.ddterm shortcut-split-horizontal @as []
# gsettings set com.github.amezin.ddterm shortcut-split-position-dec @as []
# gsettings set com.github.amezin.ddterm shortcut-split-position-inc @as []
# gsettings set com.github.amezin.ddterm shortcut-split-vertical @as []
gsettings set com.github.amezin.ddterm shortcut-switch-to-tab-1 "['<Alt>1']"
gsettings set com.github.amezin.ddterm shortcut-switch-to-tab-10 "['<Alt>0']"
gsettings set com.github.amezin.ddterm shortcut-switch-to-tab-2 "['<Alt>2']"
gsettings set com.github.amezin.ddterm shortcut-switch-to-tab-3 "['<Alt>3']"
gsettings set com.github.amezin.ddterm shortcut-switch-to-tab-4 "['<Alt>4']"
gsettings set com.github.amezin.ddterm shortcut-switch-to-tab-5 "['<Alt>5']"
gsettings set com.github.amezin.ddterm shortcut-switch-to-tab-6 "['<Alt>6']"
gsettings set com.github.amezin.ddterm shortcut-switch-to-tab-7 "['<Alt>7']"
gsettings set com.github.amezin.ddterm shortcut-switch-to-tab-8 "['<Alt>8']"
gsettings set com.github.amezin.ddterm shortcut-switch-to-tab-9 "['<Alt>9']"
gsettings set com.github.amezin.ddterm shortcut-terminal-copy "['<Ctrl><Shift>c']"
# gsettings set com.github.amezin.ddterm shortcut-terminal-copy-html @as []
gsettings set com.github.amezin.ddterm shortcut-terminal-paste "['<Ctrl><Shift>v']"
# gsettings set com.github.amezin.ddterm shortcut-terminal-reset @as []
# gsettings set com.github.amezin.ddterm shortcut-terminal-reset-and-clear @as []
# gsettings set com.github.amezin.ddterm shortcut-terminal-select-all @as []
gsettings set com.github.amezin.ddterm shortcut-toggle-maximize "['F11']"
# gsettings set com.github.amezin.ddterm shortcut-toggle-transparent-background @as []
gsettings set com.github.amezin.ddterm shortcut-win-new-tab "['<Ctrl><Shift>n']"
# gsettings set com.github.amezin.ddterm shortcut-win-new-tab-after-current @as []
# gsettings set com.github.amezin.ddterm shortcut-win-new-tab-before-current @as []
# gsettings set com.github.amezin.ddterm shortcut-win-new-tab-front @as []
# gsettings set com.github.amezin.ddterm shortcut-window-hide @as []
gsettings set com.github.amezin.ddterm shortcut-window-size-dec "['<Ctrl>Up']"
gsettings set com.github.amezin.ddterm shortcut-window-size-inc "['<Ctrl>Down']"
gsettings set com.github.amezin.ddterm shortcuts-enabled true
gsettings set com.github.amezin.ddterm show-animation 'linear'
gsettings set com.github.amezin.ddterm show-animation-duration 0.14999999999999999
gsettings set com.github.amezin.ddterm show-scrollbar true
gsettings set com.github.amezin.ddterm tab-close-buttons true
gsettings set com.github.amezin.ddterm tab-expand true
gsettings set com.github.amezin.ddterm tab-label-ellipsize-mode 'none'
gsettings set com.github.amezin.ddterm tab-label-width 0.10000000000000001
gsettings set com.github.amezin.ddterm tab-policy 'never'
gsettings set com.github.amezin.ddterm tab-position 'bottom'
gsettings set com.github.amezin.ddterm tab-show-shortcuts true
gsettings set com.github.amezin.ddterm tab-switcher-popup true
gsettings set com.github.amezin.ddterm text-blink-mode 'always'
gsettings set com.github.amezin.ddterm theme-variant 'system'
gsettings set com.github.amezin.ddterm transparent-background true
gsettings set com.github.amezin.ddterm use-system-font true
gsettings set com.github.amezin.ddterm use-theme-colors true
gsettings set com.github.amezin.ddterm window-above true
gsettings set com.github.amezin.ddterm window-maximize true
gsettings set com.github.amezin.ddterm window-monitor 'current'
gsettings set com.github.amezin.ddterm window-monitor-connector ''
gsettings set com.github.amezin.ddterm window-position 'top'
gsettings set com.github.amezin.ddterm window-resizable false
gsettings set com.github.amezin.ddterm window-size 1.0
gsettings set com.github.amezin.ddterm window-skip-taskbar true
gsettings set com.github.amezin.ddterm window-stick true

