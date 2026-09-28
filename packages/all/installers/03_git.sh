### -- Manifest
### provides: common/git
### depends_on: [common/os-libs]
### distro: [all]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  case "$DIS_DISTRO" in
    amazon_linux) sudo yum install -y git ;;
    ubuntu)       sudo DEBIAN_FRONTEND=noninteractive apt install -y git ;;
    *)            echo "common/git: unsupported distro '$DIS_DISTRO'" >&2; exit 1 ;;
  esac
fi

# git rpull: pull --rebase --autostash, keeping the upstream go.mod/go.sum on conflict
# (build tools regenerate them). The attributes are applied only by rpull.
mkdir -p ~/.config/git
cp "$DIS_CONFIG_FOLDER/git/pull-gomod-attributes" ~/.config/git/pull-gomod-attributes
git config --global alias.rpull \
  '!git -c core.attributesFile="$HOME/.config/git/pull-gomod-attributes" -c merge.keepupstream.driver=true pull --rebase --autostash'

dis tools add-rc-aliases \
  --name 'git aliases' \
  --content "$(cat <<'EOF'
alias log='git log --oneline'
alias log-e='git log'
alias commit='git commit'
alias amend='git commit --amend'
alias status='git status'
alias push='git push'

alias remove-last-commit='git reset --soft HEAD~1'
alias fuck-it='git reset --soft HEAD~1'
alias undo-last-commit='git reset --soft HEAD~1'

alias pull='git rpull'
EOF
)"
