### -- Manifest
### provides: common/bash-completion
### depends_on: [common/brew]
### distro: [amazon_linux]
### -- End
## Install bash completion (https://github.com/scop/bash-completion) from brew.
## The package for AL2 is very old and misses lots of useful completion, which the brew one contains.

brew install bash-completion@2
dis tools add-rc-init \
  --name 'Bash completion (https://github.com/scop/bash-completion)' \
  --content '# Use bash-completion, if available, and avoid double-sourcing
[[ $PS1 &&
  ! ${BASH_COMPLETION_VERSINFO:-} &&
  -f /home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh ]] &&
    . /home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh'
