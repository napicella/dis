### -- Manifest
### provides: common/aws-cli
### depends_on: [common/os-libs]
### distro: [all]
### -- End

# Set up aws cli auto complete, with v2's completer: an OS-packaged v1 ships its
# own in /usr/bin, which doesn't know v2's commands.
dis tools add-rc-init --name 'aws_completer' --content \
  'if [ -e /usr/local/bin/aws_completer ]; then
  complete -C '"'"'/usr/local/bin/aws_completer'"'"' aws
fi'

if [[ -n "${DIS_INSTALL:-}" ]]; then
  # Check v2 itself: an OS-packaged v1 aws on PATH doesn't count.
  if /usr/local/bin/aws --version 2>&1 | grep -q '^aws-cli/2\.'; then
    echo "aws cli v2 is installed"
    exit 0
  fi

  curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "/tmp/awscliv2.zip"
  unzip -o /tmp/awscliv2.zip -d /tmp
  sudo /tmp/aws/install --update
fi
