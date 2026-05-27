### -- Manifest
### provides: common/aws-cli
### depends_on: [common/os-libs]
### distro: [all]
### -- End


# set up aws cli auto complete
bashrc_init_add "aws_completer" \
"if [ -e /usr/bin/aws_completer ]; then
  complete -C '/usr/bin/aws_completer' aws
fi"

if command -v aws &> /dev/null
then
    echo "aws cli is installed"
    exit 0
fi

curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "/tmp/awscliv2.zip"
unzip /tmp/awscliv2.zip -d /tmp
sudo /tmp/aws/install