### -- Manifest
### provides: common/aws-ssm-plugin
### depends_on: [common/aws-cli]
### distro: [ubuntu]
### -- End

source $DIS_BINDING

if command -v session-manager-plugin &> /dev/null
then
    echo "aws ssm plugin is installed"
    exit 0
fi

curl "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/ubuntu_64bit/session-manager-plugin.deb" -o "/tmp/session-manager-plugin.deb"
sudo dpkg -i /tmp/session-manager-plugin.deb
rm /tmp/session-manager-plugin.deb