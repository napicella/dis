### -- Manifest
### provides: common/aws-cli
### depends_on: [common/os-libs]
### distro: [all]
### -- End

if [[ -n "${DIS_INSTALL:-}" ]]; then
  if command -v aws &> /dev/null; then
    echo "aws cli is installed"
    exit 0
  fi

  curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "/tmp/awscliv2.zip"
  unzip /tmp/awscliv2.zip -d /tmp
  sudo /tmp/aws/install

  # Set up aws cli auto complete.
  dis tools add-rc-init --name 'aws_completer' --content \
    'if [ -e /usr/bin/aws_completer ]; then
  complete -C '"'"'/usr/bin/aws_completer'"'"' aws
fi'
fi
