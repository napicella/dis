### -- Manifest
### provides: common/tmux
### depends_on: [common/brew]
### distro: [amazon_linux]
### -- End

# Amazon Linux 2 ships tmux 1.8, which rejects most of .tmux.conf (copy-mode-vi key
# table, mouse, *-style options), so install a current one from brew.
if [[ -n "${DIS_INSTALL:-}" ]]; then
  brew install tmux
fi

# Config: always re-deploy the tmux config file. It is shared with the Ubuntu
# installer, so it lives in the all/ configs.
cp "$DIS_PKG_ROOT/../all/configs/.tmux.conf" ~/
