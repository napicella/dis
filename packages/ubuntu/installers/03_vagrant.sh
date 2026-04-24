### -- Manifest
### provides: common/vagrant
### depends_on: []
### distro: [ubuntu]
### -- End

source $DIS_BINDING

if command -v vagrant &> /dev/null
then
    echo "vagrant is installed"
    exit 0
fi

# Install vagrant
wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt -y update && sudo apt install -y vagrant
vagrant plugin install vagrant-docker-compose